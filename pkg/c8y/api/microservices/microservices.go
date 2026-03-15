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
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core/artifact"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
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
	app := model.Microservice{}

	// set default name to the file name
	fileExt := filepath.Ext(opt.File)
	appNameFromFile := artifact.ParseName(opt.File)

	// Set application properties

	if strings.HasSuffix(opt.File, ".zip") {
		// Try loading manifest file directly from the zip (without unzipping it)
		slog.Info("Trying to detect manifest from a zip file", "path", opt.File)
		if err := GetManifestContents(opt.File, &app.Manifest); err != nil {
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

		if err := json.Unmarshal(byteValue, &app.Manifest); err != nil {
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

func (s *Service) CreateOrUpdate(ctx context.Context, opt CreateOptions) (*model.Microservice, error) {
	var application *model.Microservice
	var applicationID string
	var applicationName string

	applicationDetails, err := getApplicationDetails(opt)
	if err != nil {
		return applicationDetails, err
	}

	if applicationDetails != nil {
		applicationName = applicationDetails.Name
	}

	if applicationName == "" {
		return nil, fmt.Errorf("could not detect application name for the given input")
	}

	if applicationName != "" {
		// Only lookup microservices in the current tenant, as managing microservices of subtenants is not allowed
		// e.g. upload binary etc. Restricting the search means name conflicts will be avoided if
		// subtenants also have the same application name deployed multiple times.
		result, found := s.FindFirst(ctx, ListOptions{
			Name: applicationName,
		})
		if result.Err != nil {
			return nil, result.Err
		}
		if !found {
			return nil, core.ErrNotFound("microservice %q not found", applicationName)
		}
		if result.Data.Exists("id") {
			applicationID = result.Data.ID()
		}
	}

	if applicationID == "" {
		// Create the application
		slog.Info("Creating new application")
		result := s.Create(ctx, applicationDetails)
		if result.Err != nil {
			return nil, fmt.Errorf("failed to create microservice. %w", result.Err)
		}
		application = &model.Microservice{
			ID:   result.Data.ID(),
			Name: result.Data.Name(),
			Key:  result.Data.Key(),
		}
	} else {
		// Get existing application
		slog.Info("Getting existing application", "id", applicationID)
		result := s.Get(context.Background(), applicationID)

		if result.Err != nil {
			return nil, fmt.Errorf("failed to get microservice. %w", result.Err)
		}
		application = &model.Microservice{
			ID:   result.Data.ID(),
			Name: result.Data.Name(),
			Key:  result.Data.Key(),
		}
	}

	skipUpload := opt.SkipUpload

	if _, err := os.Stat(opt.File); err != nil {
		return nil, fmt.Errorf("could not read manifest file. %s. error=%s", opt.File, err)
	}

	// Only upload zip files
	if !strings.HasSuffix(opt.File, ".zip") {
		slog.Info("Skipping microservice binary upload")
		skipUpload = true
	}

	// Upload binary
	if !skipUpload {
		slog.Info("uploading microservice binary", "id", application.ID)
		result := s.Upload(ctx, application.ID, UploadFileOptions{
			FilePath: opt.File,
		})
		if result.Err != nil {
			return nil, fmt.Errorf("failed to upload file. path=%s. %w", opt.File, result.Err)
		}
	} else {
		//
		// Upload information from the cumulocity manifest file
		// because the zip file is not being uploaded because the app
		// will be hosted outside of the platform
		//
		// Read the Cumulocity.json file, and upload
		slog.Info(
			"Updating application details",
			"id",
			application.ID,
			"requiredRoles",
			strings.Join(applicationDetails.Manifest.RequiredRoles, ","),
			"roles",
			strings.Join(applicationDetails.Manifest.Roles, ","),
		)
		result := s.Update(ctx, application.ID, &model.Microservice{
			RequiredRoles: applicationDetails.Manifest.RequiredRoles,
			Roles:         applicationDetails.Manifest.Roles,
		})
		if result.Err != nil {
			return application, result.Err
		}
	}

	// App subscription
	if !opt.SkipSubscription {
		slog.Info("Subscribing to microservice")
		result := s.Subscribe(ctx, opt.TenantID, application.Self)
		if core.ErrHasStatus(result.Err, 409) {
			slog.Info("microservice is already subscribed to")
			result.Err = nil
		}
		if result.Err != nil {
			return application, fmt.Errorf("failed to subscribe to application. %w", result.Err)
		}
	}
	return application, nil
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
