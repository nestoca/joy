package diagnostics

import (
	"fmt"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/acarl005/stripansi"

	"github.com/nestoca/joy/internal/config"
	"github.com/nestoca/joy/internal/dependencies"
)

func Evaluate(cliVersion string, cfg *config.Config) Groups {
	return Groups{
		diagnoseExecutable(cfg, cliVersion, ExecutableOptions{}),
		diagnoseDependencies(dependencies.AllRequired, dependencies.AllOptional),
		diagnoseConfig(cfg, ConfigOpts{}),
		diagnoseCatalog(cfg.CatalogDir, CatalopOpts{}),
	}
}

const (
	success = "success"
	warning = "warning"
	failed  = "failed"
	hint    = "hint"
	info    = "info"
)

type Message struct {
	Type    string
	Value   string
	Details Messages
}

func (msg Message) StripAnsi() Message {
	msg.Value = stripansi.Strip(msg.Value)
	for i, detail := range msg.Details {
		msg.Details[i] = detail.StripAnsi()
	}
	return msg
}

func (msg Message) String() string {
	emoji := func() string {
		switch msg.Type {
		case success:
			return "✅"
		case warning:
			return "⚠️"
		case failed:
			return "💔"
		case hint:
			return "👉"
		case info:
			fallthrough
		default:
			return "➡️"
		}
	}()
	return emoji + " " + msg.Value + "\n" + indent(msg.Details.String())
}

type Messages []Message

func (set Messages) String() string {
	var builder strings.Builder
	for _, msg := range set {
		builder.WriteString(msg.String())
	}
	return builder.String()
}

type Group struct {
	Title     string
	Messages  Messages
	SubGroups Groups

	toplevel bool
}

func (group *Group) AddMsg(typ string, value string, details ...Message) *Group {
	group.Messages = append(group.Messages, msg(typ, value, details...))
	return group
}

func (group *Group) AddSubGroup(sub ...Group) *Group {
	group.SubGroups = append(group.SubGroups, sub...)
	return group
}

type Stats struct {
	Failed   int
	Warnings int
}

func (group Group) Stats() (result Stats) {
	for _, msg := range group.Messages {
		switch msg.Type {
		case failed:
			result.Failed++
		case warning:
			result.Warnings++
		}
	}

	for _, sub := range group.SubGroups {
		stats := sub.Stats()
		result.Failed += stats.Failed
		result.Warnings += stats.Warnings
	}

	return
}

func (group Group) StripAnsi() Group {
	group.Title = stripansi.Strip(group.Title)
	for i, msg := range group.Messages {
		group.Messages[i] = msg.StripAnsi()
	}
	for i, sub := range group.SubGroups {
		group.SubGroups[i] = sub.StripAnsi()
	}
	return group
}

func (group Group) String() string {
	title := func() string {
		if !group.toplevel {
			return color.InBold(color.InBlue(group.Title))
		}
		emoji := func() string {
			stats := group.Stats()
			switch {
			case stats.Failed > 0:
				return "💔"
			case stats.Warnings > 0:
				return "⚠️"
			default:
				return "✅"
			}
		}()
		return emoji + " " + color.InBold(group.Title)
	}()

	return title + "\n" + indent(group.Messages.String()+group.SubGroups.String())
}

type Groups []Group

func (groups Groups) String() string {
	if len(groups) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, group := range groups {
		builder.WriteString(group.String())
		if i != len(groups)-1 {
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func (groups Groups) Stats() (stats Stats) {
	for _, group := range groups {
		groupStats := group.Stats()
		stats.Failed += groupStats.Failed
		stats.Warnings += groupStats.Warnings
	}
	return
}

func indent(value string) string {
	if value == "" {
		return ""
	}
	return "  " + strings.ReplaceAll(value, "\n", "\n  ")
}

func msg(typ string, value string, details ...Message) Message {
	return Message{
		Type:    typ,
		Value:   value,
		Details: details,
	}
}

func label(label string, value any) string {
	return fmt.Sprintf("%s %v", color.InBold(label+":"), value)
}
