package app

// optin.go holds what the opt-in outbound features share between their config
// and their use-cases (P-D9, P-D12).

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

func optInChoice(choice config.OptInChoice) outbound.Choice {
	switch choice {
	case config.OptInOn:
		return outbound.ChoiceOn
	case config.OptInOff:
		return outbound.ChoiceOff
	case config.OptInUnanswered:
	}
	return outbound.ChoiceUnanswered
}

// optInAnswer stores an operator's answer, which Validate has limited to on
// or off.
func optInAnswer(choice outbound.Choice) config.OptInChoice {
	if choice == outbound.ChoiceOn {
		return config.OptInOn
	}
	return config.OptInOff
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func emptyIfDefault(value, fallback string) string {
	if value == fallback {
		return ""
	}
	return value
}
