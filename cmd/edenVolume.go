package cmd

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	volumeName string
)

var volumeCmd = &cobra.Command{
	Use: "volume",
}

//volumeLsCmd is a command to list deployed volumes
var volumeLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List volumes",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		state := eve.Init(ctrl, dev)
		if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := state.VolumeList(); err != nil {
			log.Fatal(err)
		}
	},
}

//volumeCreateCmd is a command to create volume
var volumeCreateCmd = &cobra.Command{
	Use:   "create <(docker|http(s)|file)://(<TAG>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>)>",
	Short: "Create volume",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appLink := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		var opts []expect.ExpectationOption
		diskSizeParsed, err := humanize.ParseBytes(diskSize)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))
		registryToUse := registry
		switch registry {
		case "local":
			registryToUse = fmt.Sprintf("%s:%d", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
		case "remote":
			registryToUse = ""
		}
		opts = append(opts, expect.WithRegistry(registryToUse))
		expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, volumeName, opts...)
		volumeConfig := expectation.Volume()
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
		log.Infof("create volume %s with %s request sent", volumeConfig.DisplayName, appLink)
	},
}

//volumeDeleteCmd is a command to delete volume
var volumeDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete volume",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		volumeName = args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for id, el := range dev.GetVolumes() {
			volume, err := ctrl.GetVolume(el)
			if err != nil {
				log.Fatalf("no volume in cloud %s: %s", el, err)
			}
			if volume.DisplayName == volumeName {
				configs := dev.GetVolumes()
				utils.DelEleInSlice(&configs, id)
				dev.SetVolumeConfigs(configs)
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("volume %s delete done", volumeName)
				return
			}
		}
		log.Infof("not found volume with name %s", volumeName)
	},
}

func volumeInit() {
	volumeCmd.AddCommand(volumeLsCmd)

	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCreateCmd.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	volumeCreateCmd.Flags().StringVar(&diskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	volumeCreateCmd.Flags().StringVarP(&volumeName, "name", "n", "", "name of volume, random if empty")

	volumeCmd.AddCommand(volumeDeleteCmd)
}
