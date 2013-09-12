package commands

import (
	"cf/api"
	"cf/requirements"
	term "cf/terminal"
	"github.com/codegangsta/cli"
)

type Spaces struct {
	ui        term.UI
	spaceRepo api.SpaceRepository
}

func NewSpaces(ui term.UI, spaceRepo api.SpaceRepository) (cmd Spaces) {
	cmd.ui = ui
	cmd.spaceRepo = spaceRepo
	return
}

func (cmd Spaces) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	reqs = []requirements.Requirement{
		reqFactory.NewLoginRequirement(),
		reqFactory.NewOrgRequirement(),
	}
	return
}

func (cmd Spaces) Run(c *cli.Context) {
	cmd.ui.Say("Getting spaces...")

	spaces, err := cmd.spaceRepo.FindAll()
	if err != nil {
		cmd.ui.Failed(err.Error())
		return
	}

	cmd.ui.Ok()

	for _, space := range spaces {
		cmd.ui.Say(space.Name)
	}
}