package converter

import "github.com/bwmarrin/discordgo"

// Options that can be set when creatng a backup
type OldBackupCreateOpts struct {
	Channels                  []string       `description:"If set, the channels to prune messages from"`
	PerChannel                int            `description:"The number of messages per channel"`
	MaxMessages               int            `description:"The maximum number of messages to backup"`
	BackupMessages            bool           `description:"Whether to backup messages or not"`
	BackupGuildAssets         []string       `description:"What assets to back up"`
	IgnoreMessageBackupErrors bool           `description:"Whether to ignore errors while backing up messages or not and skip these channels"`
	RolloverLeftovers         bool           `description:"Whether to attempt rollover of leftover message quota to another channels or not"`
	SpecialAllocations        map[string]int `description:"Specific channel allocation overrides"`
}

// Represents a backed up message
type BackupMessage struct {
	Message *discordgo.Message `json:"message"`
}

func array[T any](v []T) []T {
	if len(v) == 0 {
		return []T{} // Ensure we return an empty slice, not nil
	}

	return v
}

func hashmap[K comparable, V any](v map[K]V) map[K]V {
	if len(v) == 0 {
		return map[K]V{} // Ensure we return an empty map, not nil
	}

	return v
}

func (opts *OldBackupCreateOpts) ToNew() BackupCreateOpts {
	// Remove any assets not 'icon', 'banner', or 'splash' from backupGuildAssets
	var validAssets = []string{}

	for _, asset := range opts.BackupGuildAssets {
		switch asset {
		case "guildIcon":
			validAssets = append(validAssets, "icon")
		case "guildBanner":
			validAssets = append(validAssets, "banner")
		case "guildSplash":
			validAssets = append(validAssets, "splash")
		}
	}

	return BackupCreateOpts{
		Channels:           array(opts.Channels),
		PerChannel:         opts.PerChannel,
		MaxMessages:        opts.MaxMessages,
		BackupMessages:     opts.BackupMessages,
		BackupGuildAssets:  array(validAssets),
		SpecialAllocations: hashmap(opts.SpecialAllocations),
	}
}
