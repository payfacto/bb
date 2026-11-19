package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/internal/sshhost"
	"github.com/payfacto/bb/pkg/bitbucket"
)

const defaultSSHPort = 22

var pipelineSSHKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "Manage the SSH key pair Pipelines builds use",
}

var pipelineSSHKeyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the public key of the Pipelines SSH key pair",
	Long:  "Show the public key of the Pipelines SSH key pair. Bitbucket never returns the private key.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pair, err := client.PipelineSSHKey(ws, repo).Get(context.Background())
		if err != nil {
			return err
		}
		return printOutput(pair, func() { fmt.Println(pair.PublicKey) })
	},
}

var (
	pipelineSSHKeySetPrivateFile string
	pipelineSSHKeySetPublicFile  string
)

var pipelineSSHKeySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Create or replace the Pipelines SSH key pair from two key files",
	Long: `Create or replace the Pipelines SSH key pair.

Both keys are read from files; the private key is never accepted on the
command line and never printed. The private key must be unencrypted (no
passphrase) and the two files must belong to the same key pair.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		privateKey, publicKey, err := readKeyPairFiles(pipelineSSHKeySetPrivateFile, pipelineSSHKeySetPublicFile)
		if err != nil {
			return err
		}
		pair, err := client.PipelineSSHKey(ws, repo).Set(context.Background(), privateKey, publicKey)
		if err != nil {
			return err
		}
		return printOutput(pair, func() { fmt.Printf("Pipelines SSH key pair set. Public key:\n%s\n", pair.PublicKey) })
	},
}

// readKeyPairFiles loads and validates the two key files. Errors name the
// flag and the problem, never the key contents.
func readKeyPairFiles(privatePath, publicPath string) (privateKey, publicKey string, err error) {
	privatePEM, err := readKeyFile("private-key-file", privatePath)
	if err != nil {
		return "", "", err
	}
	publicLine, err := readKeyFile("public-key-file", publicPath)
	if err != nil {
		return "", "", err
	}
	if err := sshhost.CheckKeyPair(privatePEM, string(publicLine)); err != nil {
		return "", "", newCLIError(ErrCodeValidationFailed, err.Error(), nil)
	}
	return string(privatePEM), strings.TrimSpace(string(publicLine)), nil
}

func readKeyFile(flagName, path string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, newCLIError(ErrCodeValidationFailed, fmt.Sprintf("--%s: %s", flagName, err.Error()), err)
	}
	return contents, nil
}

var pipelineSSHKeyDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the Pipelines SSH key pair",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PipelineSSHKey(ws, repo).Delete(context.Background()); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true}, func() {
			fmt.Println("Pipelines SSH key pair deleted")
		})
	},
}

var pipelineKnownHostCmd = &cobra.Command{
	Use:   "known-host",
	Short: "Manage the SSH known hosts Pipelines builds trust",
}

var pipelineKnownHostListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the Pipelines SSH known hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		hosts, err := client.PipelineKnownHosts(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(hosts, func() {
			if len(hosts) == 0 {
				fmt.Println("No known hosts.")
				return
			}
			for _, h := range hosts {
				fmt.Printf("%s  %s  %s  %s\n", h.UUID, h.Hostname, h.PublicKey.KeyType, h.PublicKey.SHA256Fingerprint)
			}
		})
	},
}

// knownHostAddOptions holds the `known-host add` flags.
type knownHostAddOptions struct {
	hostname  string
	port      int
	publicKey string // "type base64"; empty means fetch from the host
	keyType   string // restricts the fetch; empty means the host's preference
}

var knownHostAdd knownHostAddOptions

var pipelineKnownHostAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a Pipelines SSH known host, fetching its host key unless --public-key is given",
	Long: `Add a Pipelines SSH known host.

Without --public-key, bb connects to the host over SSH, takes the host key it
presents, and prints its SHA256 fingerprint to stderr. That key is trusted on
first use: compare the fingerprint with the one the host's owner publishes, or
pass a key you already trust with --public-key "type base64".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		key, err := resolveKnownHostKey(knownHostAdd, sshhost.Fetch)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "note: %s host key %s %s\n", knownHostAdd.hostname, key.Type, key.SHA256Fingerprint)
		host, err := client.PipelineKnownHosts(ws, repo).Add(context.Background(), bitbucket.AddKnownHostInput{
			Hostname:  knownHostAdd.hostname,
			PublicKey: bitbucket.KnownHostKey{KeyType: key.Type, Key: key.Base64},
		})
		if err != nil {
			return err
		}
		return printOutput(host, func() {
			fmt.Printf("Known host added: %s %s (%s)\n", host.Hostname, host.PublicKey.KeyType, host.UUID)
		})
	},
}

// hostKeyFetcher matches sshhost.Fetch, so tests can stand in for the network.
type hostKeyFetcher func(ctx context.Context, addr, keyType string) (sshhost.HostKey, error)

// resolveKnownHostKey returns the key to register: the one given with
// --public-key, or else the one the host presents.
func resolveKnownHostKey(opts knownHostAddOptions, fetch hostKeyFetcher) (sshhost.HostKey, error) {
	if opts.publicKey == "" {
		addr := net.JoinHostPort(opts.hostname, strconv.Itoa(opts.port))
		key, err := fetch(context.Background(), addr, opts.keyType)
		if err != nil {
			return sshhost.HostKey{}, newCLIError(ErrCodeValidationFailed, err.Error(), err)
		}
		return key, nil
	}
	key, err := sshhost.ParseAuthorizedKey(opts.publicKey)
	if err != nil {
		return sshhost.HostKey{}, newCLIError(ErrCodeValidationFailed, "--public-key: "+err.Error(), err)
	}
	if opts.keyType != "" && opts.keyType != key.Type {
		return sshhost.HostKey{}, newCLIError(ErrCodeValidationFailed,
			fmt.Sprintf("--key-type %s does not match the %s key given with --public-key", opts.keyType, key.Type), nil)
	}
	return key, nil
}

var pipelineKnownHostDeleteUUID string

var pipelineKnownHostDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Pipelines SSH known host by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.PipelineKnownHosts(ws, repo).Delete(context.Background(), pipelineKnownHostDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"deleted": true, "uuid": pipelineKnownHostDeleteUUID}, func() {
			fmt.Printf("Known host %s deleted\n", pipelineKnownHostDeleteUUID)
		})
	},
}

func init() {
	pipelineSSHKeySetCmd.Flags().StringVar(&pipelineSSHKeySetPrivateFile, "private-key-file", "", "path to the unencrypted private key (required)")
	pipelineSSHKeySetCmd.Flags().StringVar(&pipelineSSHKeySetPublicFile, "public-key-file", "", "path to the matching public key, e.g. id_ed25519.pub (required)")
	pipelineSSHKeySetCmd.MarkFlagRequired("private-key-file")
	pipelineSSHKeySetCmd.MarkFlagRequired("public-key-file")
	pipelineSSHKeyCmd.AddCommand(pipelineSSHKeyGetCmd, pipelineSSHKeySetCmd, pipelineSSHKeyDeleteCmd)

	pipelineKnownHostAddCmd.Flags().StringVar(&knownHostAdd.hostname, "hostname", "", "host to trust, e.g. github.com (required)")
	pipelineKnownHostAddCmd.Flags().IntVar(&knownHostAdd.port, "port", defaultSSHPort, "SSH port used to fetch the host key")
	pipelineKnownHostAddCmd.Flags().StringVar(&knownHostAdd.publicKey, "public-key", "", `host public key as "type base64"; skips fetching it from the host`)
	pipelineKnownHostAddCmd.Flags().StringVar(&knownHostAdd.keyType, "key-type", "", "key type to fetch, e.g. ssh-ed25519 or ssh-rsa (default: the host's preferred key)")
	pipelineKnownHostAddCmd.MarkFlagRequired("hostname")
	pipelineKnownHostDeleteCmd.Flags().StringVar(&pipelineKnownHostDeleteUUID, "uuid", "", "known host UUID (required)")
	pipelineKnownHostDeleteCmd.MarkFlagRequired("uuid")
	pipelineKnownHostCmd.AddCommand(pipelineKnownHostListCmd, pipelineKnownHostAddCmd, pipelineKnownHostDeleteCmd)

	pipelineConfigCmd.AddCommand(pipelineSSHKeyCmd, pipelineKnownHostCmd)
}
