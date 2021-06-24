package fieldlabs

import (
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
)

var templates = &promptui.PromptTemplates{
	Prompt:  "{{ . | bold }} ",
	Valid:   "{{ . | green }} ",
	Invalid: "{{ . | red }} ",
	Success: "{{ . | bold }} ",
}

func PromptConfirmDelete() (string, error) {

	prompt := promptui.Prompt{
		Label:     "Delete the above listed applications? There is no undo:",
		Templates: templates,
		Default:   "",
		Validate: func(input string) error {
			// "no" will exit with a "prompt declined" error, just in case they don't think to ctrl+c
			if input == "no" || input == "yes" {
				return nil
			}
			return errors.New(`only "yes" will be accepted`)
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				return "", errors.New("interrupted")
			}
			continue
		}

		return result, nil
	}
}
