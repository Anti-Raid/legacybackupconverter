package converter

/*
The file format for backups v2:

Internally a backup is a TAR file with the .arb1 file extension. Encrypted backups are simply a AES256 encrypted ARB1 with the .arb1e file extension.

TAR File Contents:
- `core.json`: A JSON file containing the cote backup data.
- `assets/{asset_name}.jpg`: A directory containing all assets that are backed up, such as guild icons (and maybe emojis in the future?).

## Core Backup Data Format

The JSON file contains the following fields:
- `guild`: The guild object from Discord (a `discordTypes.GuildObject`)
- `channels`: The channels in the guild, as an array of `discordTypes.ChannelObject` (this is a subset of the channels that were backed up).
- `messages`: An array of messages (`discordTypes.MessageObject`).
- `options`: The options used to create the backup, as defined in `BackupCreateOpts`.
- `channel_allocation`: The final channel allocation for the backup, mapping channel IDs to the number of messages backed up in that channel.
*/

import "github.com/bwmarrin/discordgo"

type CoreBackupData struct {
	Guild             discordgo.Guild                `json:"guild"`
	Channels          []discordgo.Channel            `json:"channels"`
	Messages          map[string][]discordgo.Message `json:"messages"`
	Options           BackupCreateOpts               `json:"options"`
	ChannelAllocation map[string]int                 `json:"channel_allocation"`
}

type BackupCreateOpts struct {
	Channels           []string       `json:"channels"`
	PerChannel         int            `json:"perChannel"`
	MaxMessages        int            `json:"maxMessages"`
	BackupMessages     bool           `json:"backupMessages"`
	BackupGuildAssets  []string       `json:"backupGuildAssets"`  // "icon", "banner", "splash"
	SpecialAllocations map[string]int `json:"specialAllocations"` // Specific channel allocation overrides
}
