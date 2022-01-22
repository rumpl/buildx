package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/buildx/store"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/cli/cli/command"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	builders   []string
	keepState  bool
	keepDaemon bool
}

func runRm(dockerCli command.Cli, in rmOptions) error {
	ctx := appcontext.Context()

	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return err
	}
	defer release()

	if len(in.builders) != 0 {
		var errs []string
		for _, builder := range in.builders {
			ng, err := storeutil.GetNodeGroup(txn, dockerCli, builder)
			if err != nil {
				errs = append(errs, "Error: "+err.Error())
				continue
			}
			err1 := rm(ctx, dockerCli, ng, in.keepState, in.keepDaemon)
			if err := txn.Remove(ng.Name); err != nil {
				errs = append(errs, "Error: "+err1.Error())
				continue
			}
			fmt.Fprintln(dockerCli.Out(), builder)
		}
		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "\n"))
		}
		return nil
	}

	ng, err := storeutil.GetCurrentInstance(txn, dockerCli)
	if err != nil {
		return err
	}
	if ng != nil {
		err1 := rm(ctx, dockerCli, ng, in.keepState, in.keepDaemon)
		if err := txn.Remove(ng.Name); err != nil {
			return err
		}
		return err1
	}

	return nil
}

func rmCmd(dockerCli command.Cli, rootOpts *rootOptions) *cobra.Command {
	var options rmOptions

	cmd := &cobra.Command{
		Use:           "rm [NAME] [NAME...]",
		Short:         "Remove builder instances",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootOpts.builder != "" {
				options.builders = []string{rootOpts.builder}
			}
			if len(args) > 0 {
				options.builders = args
			}
			if err := runRm(dockerCli, options); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.keepState, "keep-state", false, "Keep BuildKit state")
	flags.BoolVar(&options.keepDaemon, "keep-daemon", false, "Keep the buildkitd daemon running")

	return cmd
}

func rm(ctx context.Context, dockerCli command.Cli, ng *store.NodeGroup, keepState, keepDaemon bool) error {
	dis, err := driversForNodeGroup(ctx, dockerCli, ng, "")
	if err != nil {
		return err
	}
	for _, di := range dis {
		if di.Driver == nil {
			continue
		}
		// Do not stop the buildkitd daemon when --keep-daemon is provided
		if !keepDaemon {
			if err := di.Driver.Stop(ctx, true); err != nil {
				return err
			}
		}
		if err := di.Driver.Rm(ctx, true, !keepState, !keepDaemon); err != nil {
			return err
		}
		if di.Err != nil {
			err = di.Err
		}
	}
	return err
}
