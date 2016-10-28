package xva

import (
	"fmt"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	xsclient "github.com/xenserver/go-xenserver-client"
)

type stepInstantiateTemplate struct {
	instance *xsclient.VM
	vdi      *xsclient.VDI
}

func (self *stepInstantiateTemplate) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(config)

	if config.SourceTemplate == "" {
		return multistep.ActionContinue
	}

	client := state.Get("client").(xsclient.XenAPIClient)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Step: Instantiate Template")

	var template *xsclient.VM
	var err error

	if len(config.SourceTemplate) >= 7 && config.SourceTemplate[:7] == "uuid://" {
		templateUuid := config.SourceTemplate[7:]

		template, err = client.GetVMByUuid(templateUuid)
		if err != nil {
			ui.Error(fmt.Sprintf("Could not get template with UUID '%s': %s", templateUuid, err.Error()))
			ui.Error("Defaulting to use \"source_path\".")
			return multistep.ActionContinue
		}
	} else {
		templates, err := client.GetVMByNameLabel(config.SourceTemplate)
		if err != nil {
			ui.Error(fmt.Sprintf("Error getting template: %s", err.Error()))
			ui.Error("Defaulting to use \"source_path\".")
			return multistep.ActionContinue
		}

		switch {
		case len(templates) == 0:
			ui.Error(fmt.Sprintf("Couldn't find a template with the name-label '%s'.", config.SourceTemplate))
			ui.Error("Defaulting to use \"source_path\".")
			return multistep.ActionContinue
		case len(templates) > 1:
			ui.Error(fmt.Sprintf("Found more than one template with the name '%s'. The name must be unique.", config.SourceTemplate))
			ui.Error("Defaulting to use \"source_path\".")
			return multistep.ActionContinue
		}

		template = templates[0]
	}

	self.instance, err = template.Clone(config.VMName)
	if err != nil {
		ui.Error(fmt.Sprintf("Error cloning template: %s", err.Error()))
		return multistep.ActionHalt
	}

	instanceId, err := self.instance.GetUuid()
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VM UUID: %s", err.Error()))
		return multistep.ActionHalt
	}
	state.Put("instance_uuid", instanceId)

	err = self.instance.SetIsATemplate(false)
	if err != nil {
		ui.Error(fmt.Sprintf("Error converting template to a VM: %s", err.Error()))
		return multistep.ActionHalt
	}

	err = self.instance.SetDescription(config.VMDescription)
	if err != nil {
		ui.Error(fmt.Sprintf("Error setting VM description: %s", err.Error()))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Instantiated template '%s'", instanceId))

	return multistep.ActionContinue
}

func (self *stepInstantiateTemplate) Cleanup(state multistep.StateBag) {}