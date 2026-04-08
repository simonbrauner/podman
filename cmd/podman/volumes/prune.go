package volumes

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containers/podman/v6/cmd/podman/common"
	"github.com/containers/podman/v6/cmd/podman/parse"
	"github.com/containers/podman/v6/cmd/podman/registry"
	"github.com/containers/podman/v6/cmd/podman/utils"
	"github.com/containers/podman/v6/cmd/podman/validate"
	"github.com/containers/podman/v6/pkg/domain/entities"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/completion"
)

var (
	volumePruneDescription = `By default, only anonymous (unnamed) unused volumes are removed.
  Use --all to remove all unused volumes (anonymous and named).
  The command prompts for confirmation which can be overridden with the --force flag.`
	pruneCommand = &cobra.Command{
		Use:               "prune [options]",
		Args:              validate.NoArgs,
		Short:             "Remove unused volumes",
		Long:              volumePruneDescription,
		RunE:              prune,
		ValidArgsFunction: completion.AutocompleteNone,
	}
	filter = []string{}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: pruneCommand,
		Parent:  volumeCmd,
	})
	flags := pruneCommand.Flags()

	filterFlagName := "filter"
	flags.StringArrayVar(&filter, filterFlagName, []string{}, "Provide filter values (e.g. 'label=<key>=<value>')")
	_ = pruneCommand.RegisterFlagCompletionFunc(filterFlagName, common.AutocompleteVolumePruneFilters)
	flags.BoolP("force", "f", false, "Do not prompt for confirmation")
	flags.BoolP("all", "a", false, "Remove all unused volumes, both anonymous and named")
}

func prune(cmd *cobra.Command, _ []string) error {
	var (
		pruneOptions = entities.VolumePruneOptions{}
		listOptions  = entities.VolumeListOptions{}
	)
	// Prompt for confirmation if --force is not set
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	pruneOptions.Filters, err = parse.FilterArgumentsIntoFilters(filter)
	if err != nil {
		return err
	}

	// --all adds filter all=true (Docker-compatible; behavior is filter-only)
	allFlag, _ := cmd.Flags().GetBool("all")
	filterAllFlag := strings.EqualFold(pruneOptions.Filters.Get("all"), "true")
	if allFlag && filterAllFlag {
		return errors.New("--all and --filter all cannot be used together")
	}
	allFlag = allFlag || filterAllFlag
	if allFlag {
		pruneOptions.Filters.Set("all", "true")
	}

	if !force {
		reader := bufio.NewReader(os.Stdin)
		if allFlag {
			fmt.Println("WARNING! This will remove all volumes not used by at least one container. The following volumes will be removed:")
		} else {
			fmt.Println("WARNING! This will remove anonymous local volumes not used by at least one container. The following volumes will be removed:")
		}
		listOptions.Filter, err = parse.FilterArgumentsIntoFilters(filter)
		if err != nil {
			return err
		}

		// List does not support all=true filter, its arguments need to be adjusted
		delete(listOptions.Filter, "all")
		listOptions.Filter["dangling"] = []string{"true"}
		if !allFlag {
			listOptions.Filter["anonymous"] = []string{"true"}
		}

		filteredVolumes, err := registry.ContainerEngine().VolumeList(context.Background(), listOptions)
		if err != nil {
			return err
		}
		if len(filteredVolumes) < 1 {
			if allFlag {
				fmt.Println("No dangling volumes found")
			} else {
				fmt.Println("No dangling anonymous volumes found")
			}
			return nil
		}
		for _, fv := range filteredVolumes {
			fmt.Println(fv.Name)
		}
		fmt.Print("Are you sure you want to continue? [y/N] ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.ToLower(answer)[0] != 'y' {
			return nil
		}
	}
	responses, err := registry.ContainerEngine().VolumePrune(context.Background(), pruneOptions)
	if err != nil {
		return err
	}
	return utils.PrintVolumePruneResults(responses, false)
}
