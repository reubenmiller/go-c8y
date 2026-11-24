package microservices

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
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

// Deploy creates a microservice, or updates the existing one with the same
// name in the current tenant, from a zip file or a manifest (cumulocity.json):
//
//  1. Look for the microservice by name; create it when none exists.
//  2. For a zip file, upload it as the microservice's binary. For any other
//     file the microservice is hosted outside of the platform, so only the
//     roles from the manifest are updated.
//  3. Subscribe the tenant (opt.TenantID, default the client's tenant)
//     unless SkipSubscription is set. An existing subscription is not an
//     error.
//
// The result is the microservice. Its Status is StatusCreated for a new
// microservice and StatusUpdated for an existing one. In a dry run the lookup
// is still sent, so the result shows whether the microservice would be
// created, and the later steps are skipped.
func (s *Service) Deploy(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Microservice] {
	return op.ExecuteOrDefer(ctx, op.WithMeta(op.WithExecutor(op.Result[jsonmodels.Microservice]{}, func(execCtx context.Context) op.Result[jsonmodels.Microservice] {
		return s.deploy(execCtx, opt)
	}), "operation", "deployMicroservice"))
}

func (s *Service) deploy(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Microservice] {
	if _, err := os.Stat(opt.File); err != nil {
		return op.Failed[jsonmodels.Microservice](fmt.Errorf("could not read manifest file. path=%s. %w", opt.File, err), false)
	}
	details, err := getApplicationDetails(opt)
	if err != nil {
		return op.Failed[jsonmodels.Microservice](err, false)
	}
	if details == nil || details.Name == "" {
		return op.Failed[jsonmodels.Microservice](errors.New("could not detect the microservice name. Set the name explicitly"), false)
	}

	app, created := s.findOrCreate(ctx, details)
	if app.Err != nil || ctxhelpers.IsDryRun(ctx) {
		return app
	}
	id := app.Data.ID()

	result := app
	if !opt.SkipUpload && strings.HasSuffix(opt.File, ".zip") {
		slog.Info("uploading microservice binary", "id", id)
		upload := s.Upload(ctx, id, UploadFileOptions{FilePath: opt.File})
		if upload.Err != nil {
			return op.Failed[jsonmodels.Microservice](fmt.Errorf("failed to upload file. path=%s. %w", opt.File, upload.Err), false)
		}
	} else {
		slog.Info("updating microservice roles", "id", id,
			"requiredRoles", strings.Join(details.Manifest.RequiredRoles, ","),
			"roles", strings.Join(details.Manifest.Roles, ","))
		result = s.Update(ctx, id, &model.Microservice{
			RequiredRoles: details.Manifest.RequiredRoles,
			Roles:         details.Manifest.Roles,
		})
		if result.Err != nil {
			return result
		}
	}

	if !opt.SkipSubscription {
		slog.Info("subscribing to microservice", "id", id)
		if sub := s.Subscribe(ctx, opt.TenantID, app.Data.Self()); sub.Err != nil {
			return op.Failed[jsonmodels.Microservice](fmt.Errorf("failed to subscribe to the microservice. %w", sub.Err), false)
		}
	}

	if created {
		result.Status = op.StatusCreated
	} else {
		result.Status = op.StatusUpdated
	}
	return result
}

// findOrCreate returns the microservice with the name in the current tenant,
// or creates it; the boolean is true when it was created. Only the current
// tenant is searched, as managing microservices of subtenants (e.g. uploading
// binaries) is not allowed. The lookup is sent also in a dry run.
func (s *Service) findOrCreate(ctx context.Context, details *model.Microservice) (op.Result[jsonmodels.Microservice], bool) {
	lookupCtx := ctxhelpers.ResolutionContext(ctx)
	found, exists := s.FindFirst(lookupCtx, ListOptions{Name: details.Name})
	if found.Err != nil {
		return found, false
	}
	if exists && found.Data.Exists("id") {
		existing := s.Get(lookupCtx, found.Data.ID())
		if existing.Err != nil {
			existing.Err = fmt.Errorf("failed to get microservice. %w", existing.Err)
		}
		return existing, false
	}
	created := s.Create(ctx, details)
	if created.Err != nil {
		created.Err = fmt.Errorf("failed to create microservice. %w", created.Err)
	}
	return created, true
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
