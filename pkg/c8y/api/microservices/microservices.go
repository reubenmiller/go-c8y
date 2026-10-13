package microservices

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core/artifact" //nolint:misspell
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
)

type CreateOptions struct {
	File             string
	Name             string
	Key              string
	Availability     string
	ContextPath      string
	ResourceURL      string
	TenantID         string
	SkipSubscription bool
	SkipUpload       bool
}

func getApplicationDetails(opt CreateOptions) (*model.Microservice, error) {
	// Manifest is a pointer: always allocate it so a missing/invalid manifest does not panic
	app := model.Microservice{Manifest: &model.MicroserviceManifest{}}

	// set default name to the file name
	fileExt := filepath.Ext(opt.File)
	appNameFromFile := artifact.ParseName(opt.File) //nolint:misspell

	// Set application properties

	if strings.HasSuffix(opt.File, ".zip") {
		// Try loading manifest file directly from the zip (without unzipping it)
		slog.Info("Trying to detect manifest from a zip file", "path", opt.File)
		if err := GetManifestContents(opt.File, app.Manifest); err != nil {
			slog.Warn(fmt.Sprintf("Could not find manifest file. Expected %s to contain %s", opt.File, ManifestFile), "err", err)
		}
	} else if opt.File != "" {
		// Assume json (regardless of file type)
		slog.Info("Assuming file is json (regardless of file extension)", "path", opt.File)
		jsonFile, err := os.Open(opt.File)
		if err != nil {
			return nil, err
		}
		byteValue, _ := io.ReadAll(jsonFile)

		if err := json.Unmarshal(byteValue, app.Manifest); err != nil {
			slog.Warn("invalid manifest file. Only json or zip files are accepted", "err", strings.TrimSpace(err.Error()))
		}
	}

	// Set application name using the following preferences (first match wins)
	// 1. Explicit name
	// 2. Name from file (if the given file is not a json file) - as this allows
	//    overriding the app name by just changing the file name (and not requiring to edit it)
	// 3. Name from manifest file
	if app.Manifest.Name != "" {
		app.Name = app.Manifest.Name
	}

	if !strings.EqualFold(fileExt, ".json") && strings.EqualFold(fileExt, ".zip") {
		app.Name = appNameFromFile
	}

	if opt.Name != "" {
		app.Name = opt.Name
	}

	app.Key = app.Name
	if opt.Key != "" {
		app.Key = opt.Key
	}

	app.Type = applications.TypeMicroservice

	if opt.Availability != "" {
		app.Availability = opt.Availability
	}

	app.ContextPath = app.Name
	if opt.ContextPath != "" {
		app.ContextPath = opt.ContextPath
	}

	app.ResourcesURL = "/"
	if opt.ResourceURL != "" {
		app.ResourcesURL = opt.ResourceURL
	}
	return &app, nil
}

// CreateOrUpdate creates the microservice application when none with the same
// name exists in the current tenant, then uploads the binary (zip files only)
// or updates the roles from the manifest, and optionally subscribes the tenant.
func (s *Service) CreateOrUpdate(ctx context.Context, opt CreateOptions) (*model.Microservice, error) {
	applicationDetails, err := getApplicationDetails(opt)
	if err != nil {
		return applicationDetails, err
	}
	if applicationDetails == nil || applicationDetails.Name == "" {
		return nil, fmt.Errorf("could not detect application name for the given input")
	}

	application, err := s.findOrCreate(ctx, applicationDetails)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(opt.File); err != nil {
		return nil, fmt.Errorf("could not read manifest file. %s. error=%s", opt.File, err)
	}

	if err := s.deploy(ctx, application, applicationDetails, opt); err != nil {
		return application, err
	}

	if !opt.SkipSubscription {
		if err := s.subscribeIgnoreExisting(ctx, opt.TenantID, application.Self); err != nil {
			return application, err
		}
	}
	return application, nil
}

// findOrCreate returns the application with the given name in the current
// tenant, creating it when it does not exist. Only the current tenant is
// searched, as managing microservices of subtenants (e.g. uploading binaries)
// is not allowed, which also avoids name conflicts with subtenant applications.
func (s *Service) findOrCreate(ctx context.Context, details *model.Microservice) (*model.Microservice, error) {
	result, exists := s.FindFirst(ctx, ListOptions{Name: details.Name})
	if result.Err != nil {
		return nil, result.Err
	}

	if exists && result.Data.Exists("id") {
		applicationID := result.Data.ID()
		slog.Info("Getting existing application", "id", applicationID)
		existing := s.Get(ctx, applicationID)
		if existing.Err != nil {
			return nil, fmt.Errorf("failed to get microservice. %w", existing.Err)
		}
		return toMicroservice(existing.Data), nil
	}

	slog.Info("Creating new application")
	created := s.Create(ctx, details)
	if created.Err != nil {
		return nil, fmt.Errorf("failed to create microservice. %w", created.Err)
	}
	return toMicroservice(created.Data), nil
}

func toMicroservice(app jsonmodels.Microservice) *model.Microservice {
	return &model.Microservice{
		ID:   app.ID(),
		Name: app.Name(),
		Key:  app.Key(),
		Self: app.Self(),
	}
}

// deploy uploads the binary for zip files. For any other input the app is
// hosted outside of the platform, so only the roles from the manifest
// (cumulocity.json) are updated.
func (s *Service) deploy(ctx context.Context, application, details *model.Microservice, opt CreateOptions) error {
	if !opt.SkipUpload && strings.HasSuffix(opt.File, ".zip") {
		slog.Info("uploading microservice binary", "id", application.ID)
		result := s.Upload(ctx, application.ID, UploadFileOptions{FilePath: opt.File})
		if result.Err != nil {
			return fmt.Errorf("failed to upload file. path=%s. %w", opt.File, result.Err)
		}
		return nil
	}

	slog.Info(
		"Updating application details",
		"id", application.ID,
		"requiredRoles", strings.Join(details.Manifest.RequiredRoles, ","),
		"roles", strings.Join(details.Manifest.Roles, ","),
	)
	result := s.Update(ctx, application.ID, &model.Microservice{
		RequiredRoles: details.Manifest.RequiredRoles,
		Roles:         details.Manifest.Roles,
	})
	return result.Err
}

// subscribeIgnoreExisting subscribes the tenant to the application, treating an
// existing subscription (409) as success.
func (s *Service) subscribeIgnoreExisting(ctx context.Context, tenantID, self string) error {
	slog.Info("Subscribing to microservice")
	result := s.Subscribe(ctx, tenantID, self)
	if core.ErrHasStatus(result.Err, 409) {
		slog.Info("microservice is already subscribed to")
		return nil
	}
	if result.Err != nil {
		return fmt.Errorf("failed to subscribe to application. %w", result.Err)
	}
	return nil
}

func GetManifestContents(zipFilename string, contents any) error {
	reader, err := zip.OpenReader(zipFilename)
	if err != nil {
		return err
	}

	defer reader.Close()

	for _, file := range reader.File {
		// check if the file matches the name for application portfolio xml
		if strings.EqualFold(file.Name, ManifestFile) {
			rc, err := file.Open()
			if err != nil {
				return err
			}

			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return err
			}

			defer rc.Close()
			if err := json.Unmarshal(buf.Bytes(), &contents); err != nil {
				return err
			}
		}
	}
	return nil
}
