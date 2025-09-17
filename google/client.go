package google

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Google struct {
	Client       *http.Client
	DriveService *drive.Service
	DocsService  *docs.Service
}

type Config struct {
	ClientID          string
	ClientEmail       string
	ClientX509CertURL string
	PrivateKey        string
	PrivateKeyID      string
	ProjectID         string
}

// NewGoogleDrive creates a new GoogleDrive client
func NewGoogleDrive(config Config) (*Google, error) {
	jwtConfig := &jwt.Config{
		Email:        config.ClientEmail,
		PrivateKey:   []byte(config.PrivateKey),
		PrivateKeyID: config.PrivateKeyID,
		Scopes:       []string{drive.DriveReadonlyScope},
		TokenURL:     google.JWTTokenURL,
	}

	tokenSource := jwtConfig.TokenSource(context.Background())
	client := oauth2.NewClient(context.Background(), tokenSource)
	driveService, err := drive.NewService(context.Background(), option.WithHTTPClient(client))

	if err != nil {
		return nil, err
	}

	docsService, err := docs.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// verify the connection and credentials
	_, err = driveService.About.Get().Fields("user").Do()

	if err != nil {
		return nil, err
	}

	return &Google{
		Client:       client,
		DriveService: driveService,
		DocsService:  docsService,
	}, nil
}

// NewGoogleDriveWithWriteAccess creates a new GoogleDrive client with write permissions
func NewGoogleDriveWithWriteAccess(config Config) (*Google, error) {
	jwtConfig := &jwt.Config{
		Email:        config.ClientEmail,
		PrivateKey:   []byte(config.PrivateKey),
		PrivateKeyID: config.PrivateKeyID,
		Scopes:       []string{drive.DriveScope},
		TokenURL:     google.JWTTokenURL,
	}

	tokenSource := jwtConfig.TokenSource(context.Background())
	client := oauth2.NewClient(context.Background(), tokenSource)
	driveService, err := drive.NewService(context.Background(), option.WithHTTPClient(client))

	if err != nil {
		return nil, err
	}

	docsService, err := docs.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// verify the connection and credentials
	_, err = driveService.About.Get().Fields("user").Do()

	if err != nil {
		return nil, err
	}

	return &Google{
		Client:       client,
		DriveService: driveService,
		DocsService:  docsService,
	}, nil
}

// QueryOptions defines the options for querying Drive resources
type QueryOptions struct {
	Query                     string
	Fields                    string
	SupportsAllDrives         bool
	IncludeItemsFromAllDrives bool
}
