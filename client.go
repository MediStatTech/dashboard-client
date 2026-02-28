package client

import (
	"fmt"

	"github.com/MediStatTech/dashboard-client/client_options"
	services_v1 "github.com/MediStatTech/dashboard-client/pb/go/services/v1"
	log "github.com/MediStatTech/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	xdscreds "google.golang.org/grpc/credentials/xds"
	_ "google.golang.org/grpc/xds" // Import XDS support for production service discovery
)

type Facade struct {
	conn *grpc.ClientConn
	log  *log.Logger

	Auth    services_v1.AuthServiceClient
	Staff   services_v1.StaffServiceClient
	Patient services_v1.PatientServiceClient
	Diseas  services_v1.DiseasServiceClient
}

func New(
	o *client_options.Options,
) (*Facade, error) {
	var target string

	// Check if custom address is provided
	if o.AddressName != "" {
		target = o.AddressName
	} else {
		// XDS service discovery for production
		target = "todo-service.svc.cluster.local:8443"
	}

	// Connection options
	dialOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	}

	if o.ENV != nil && o.ENV.IsDev() {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else if o.AddressName != "" && (len(o.AddressName) < 6 || o.AddressName[:6] != "xds://") {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		creds, err := xdscreds.NewClientCredentials(xdscreds.ClientOptions{
			FallbackCreds: insecure.NewCredentials(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create xds credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		o.Log.Error("Failed to connect to dashboard service", map[string]interface{}{
			"target": target,
			"error":  err.Error(),
		})
		return nil, err
	}

	o.Log.Info("Connected to dashboard service", map[string]interface{}{
		"target": target,
	})

	return &Facade{
		conn:    conn,
		log:     o.Log,
		Auth:    services_v1.NewAuthServiceClient(conn),
		Staff:   services_v1.NewStaffServiceClient(conn),
		Patient: services_v1.NewPatientServiceClient(conn),
		Diseas:  services_v1.NewDiseasServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Facade) Close() error {
	return c.conn.Close()
}
