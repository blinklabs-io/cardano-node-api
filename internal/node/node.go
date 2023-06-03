package node

import (
	"fmt"
	"os"

	"github.com/blinklabs-io/gouroboros"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
)

func GetConnection() (*ouroboros.Connection, error) {
	cfg := config.GetConfig()
	// Connect to cardano-node
	oConn, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(uint32(cfg.Node.NetworkMagic)),
		ouroboros.WithNodeToNode(false),
	)
	if err != nil {
		return nil, fmt.Errorf("failure creating Ouroboros connection: %s", err)
	}

	if cfg.Node.Address != "" && cfg.Node.Port > 0 {
		// Connect to TCP port
		if err := oConn.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Node.Address, cfg.Node.Port)); err != nil {
			return nil, fmt.Errorf("failure connecting to node via TCP: %s", err)
		}
	} else if cfg.Node.SocketPath != "" {
		// Check that node socket path exists
		if _, err := os.Stat(cfg.Node.SocketPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("node socket path does not exist: %s", cfg.Node.SocketPath)
			} else {
				return nil, fmt.Errorf("unknown error checking if node socket path exists: %s", err)
			}
		}
		if err := oConn.Dial("unix", cfg.Node.SocketPath); err != nil {
			return nil, fmt.Errorf("failure connecting to node via UNIX socket: %s", err)
		}
	} else {
		return nil, fmt.Errorf("you must specify either the UNIX socket path or the address/port for your cardano-node")
	}
	return oConn, nil
}
