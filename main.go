package main

import (
	"fmt"
	"github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/config"
	_ "github.com/GDATASoftwareAG/cloud-provider-ionoscloud/pkg/ionos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	appconfig "k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/names"
	"k8s.io/cloud-provider/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version/verflag"
	klog "k8s.io/klog/v2"
	"math/rand"
	"os"
	"time"
)

const AppName string = "ionoscloud-cloud-controller-manager"

var version string

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	ccmOptions, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	command := &cobra.Command{
		Use:  AppName,
		Long: fmt.Sprintf("%s manages vSphere cloud resources for a Kubernetes cluster.", AppName),
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}
	var controllerInitializers map[string]app.InitFunc

	fs := command.Flags()
	namedFlagSets := ccmOptions.Flags(app.ControllerNames(app.DefaultInitFuncConstructors), app.ControllersDisabledByDefault.List(), names.CCMControllerAliases(), app.AllWebhooks, app.DisabledByDefaultWebhooks)
	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), command.Name())

	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(command.OutOrStdout())
	command.SetUsageFunc(func(cmd *cobra.Command) error {
		if _, err := fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine()); err != nil {
			return err
		}
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	command.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine()); err != nil {
			return
		}
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
	logs.InitLogs()
	defer logs.FlushLogs()

	var versionFlag *pflag.Value
	pflag.CommandLine.VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "version":
			versionFlag = &flag.Value
		}
	})

	command.Run = func(cmd *cobra.Command, args []string) {
		if versionFlag != nil && (*versionFlag).String() != "false" {
			fmt.Printf("%s %s\n", AppName, version)
			os.Exit(0)
		}
		verflag.PrintAndExitIfRequested()
		cliflag.PrintFlags(cmd.Flags())

		c, err := ccmOptions.Config(app.ControllerNames(app.DefaultInitFuncConstructors), app.ControllersDisabledByDefault.List(), names.CCMControllerAliases(), app.AllWebhooks, app.DisabledByDefaultWebhooks)
		if err != nil {
			// explicitly ignore the error by Fprintf, exiting anyway
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		klog.Infof("%s version: %s", AppName, version)

		completedConfig := c.Complete()

		cloud := initializeCloud(completedConfig)
		controllerInitializers = app.ConstructControllerInitializers(app.DefaultInitFuncConstructors, completedConfig, cloud)
		webhookConfig := make(map[string]app.WebhookConfig)
		stop := initializeWatch(completedConfig)
		if err != nil {
			klog.Fatalf("fail to initialize watch on config map %s: %v\n", err)
		}
		webhookHandlers := app.NewWebhookHandlers(webhookConfig, completedConfig, cloud)

		// initialize a notifier for cloud config update

		if err := app.Run(completedConfig, cloud, controllerInitializers, webhookHandlers, stop); err != nil {
			// explicitly ignore the error by Fprintf, exiting anyway due to app error
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}

	if err := command.Execute(); err != nil {
		// ignore error by Fprintf, exit anyway due to cmd execute error
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func initializeWatch(_ *appconfig.CompletedConfig) chan struct{} {
	return make(chan struct{})
}

func initializeCloud(cfg *appconfig.CompletedConfig) cloudprovider.Interface {
	cloudConfig := cfg.ComponentConfig.KubeCloudShared.CloudProvider

	// initialize cloud provider with the cloud provider name and config file provided
	cloud, err := cloudprovider.InitCloudProvider(config.RegisteredProviderName, cloudConfig.CloudConfigFile)
	if err != nil {
		klog.Fatalf("Cloud provider could not be initialized: %v", err)
	}
	if cloud == nil {
		klog.Fatalf("Cloud provider is nil")
	}

	return cloud
}
