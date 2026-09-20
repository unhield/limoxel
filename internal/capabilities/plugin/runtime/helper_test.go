package runtime

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/unhield/limoxel/plugin"
)

type helperTestPlugin struct {
	initialized bool
	started     bool
	stopped     bool
}

func (p *helperTestPlugin) ID() string {
	return "org.limoxel.test"
}

func (p *helperTestPlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{ID: "org.limoxel.test"}
}

func (p *helperTestPlugin) Init(ctx context.Context, host plugin.Host) error {
	p.initialized = true
	return nil
}

func (p *helperTestPlugin) Start(ctx context.Context) error {
	p.started = true
	return nil
}

func (p *helperTestPlugin) Stop(ctx context.Context) error {
	p.stopped = true
	return nil
}

// TestHelperProcess serves as the child OS process entrypoint for runtime process tests.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("HELPER_MODE")
	switch mode {
	case "server":
		plug := &helperTestPlugin{}
		server := NewHostServer(os.Stdin, os.Stdout, plug, nil)
		server.RegisterRoute("echo", func(payload []byte) ([]byte, error) {
			return fmt.Appendf(nil, "echo: %s", string(payload)), nil
		})
		_ = server.Serve(context.Background())
		os.Exit(0)

	case "crash":
		// Immediate abnormal exit
		os.Exit(42)

	case "hang":
		// Sleep indefinitely until killed
		time.Sleep(10 * time.Minute)
		os.Exit(0)

	default:
		os.Exit(1)
	}
}
