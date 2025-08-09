package converter

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/anti-raid/legacybackupconverter/iblfile"
	"github.com/bwmarrin/discordgo"
)

func ConvertFile(data []byte, password string) ([]byte, error) {
	var aes256src = iblfile.AES256Source{}
	var noencryptsrc = iblfile.NoEncryptionSource{}

	qblock, err := iblfile.QuickBlockParser(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var encryptor iblfile.AutoEncryptor
	switch string(qblock.Encryptor) {
	case noencryptsrc.ID():
		encryptor = noencryptsrc
	case aes256src.ID():
		if password == "" {
			return nil, errors.New("this backup is encrypted and hence requires a password to decrypt and convert")
		}
		aes256src.EncryptionKey = password
		encryptor = &aes256src
	default:
		return nil, fmt.Errorf("unknown encryptor: %s", qblock.Encryptor)
	}

	f, err := iblfile.OpenAutoEncryptedFile_FullFile(bytes.NewReader(data), encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to open autoencrypted file for conversion: %w", err)
	}

	sections, err := f.Sections()

	if err != nil {
		return nil, fmt.Errorf("failed to read sections: %w", err)
	}

	meta, err := iblfile.ParseMetadata(sections)

	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	if meta.Type != "backup.server" {
		return nil, fmt.Errorf("internal error: invalid file type: %s, please contact support for more information", meta.Type)
	}

	if meta.FormatVersion != "a1" {
		return nil, fmt.Errorf("internal error: invalid file format version: %s, please contact support for more information", meta.FormatVersion)
	}

	// TODO: See https://github.com/ARChronoVault/jobserver/blob/master/jobs/backups/types.go for conversion steps

	// 1. backup_opts
	bo, err := readMsgpackSection[OldBackupCreateOpts](f, "backup_opts")

	if err != nil {
		return nil, fmt.Errorf("failed to get backup_opts: %w", err)
	}

	// Convert to new spec
	newBo := bo.ToNew()

	// 2. core/guild (guild and channels)
	srcGuild, err := readMsgpackSection[discordgo.Guild](f, "core/guild")

	if err != nil {
		return nil, fmt.Errorf("failed to get core data: %w", err)
	}

	if srcGuild.ID == "" {
		return nil, fmt.Errorf("guild data is invalid [id is empty], likely an internal decoding error")
	}

	channels := srcGuild.Channels

	var channelsList []discordgo.Channel = make([]discordgo.Channel, 0, len(channels))
	for _, channel := range channels {
		if channel == nil || channel.ID == "" {
			continue // Skip nil or empty channels
		}
		channelsList = append(channelsList, *channel)
	}

	if len(channelsList) == 0 {
		return nil, fmt.Errorf("sanity check failed during legacy backups migration: guild has no channels")
	}

	// Trim out the big useless fields that do not even exist in the new spec
	srcGuild.Channels = nil
	srcGuild.Threads = nil
	srcGuild.Members = nil
	srcGuild.Presences = nil
	srcGuild.VoiceStates = nil

	// 3. messages
	var messagesMap = make(map[string][]discordgo.Message)
	var channelAllocations = make(map[string]int)
	for _, channel := range channelsList {
		if _, ok := sections["messages/"+channel.ID]; !ok {
			// No messages for this channel, skip it
			continue
		}

		// Read messages for this channel
		messages, err := readMsgpackSection[[]discordgo.Message](f, "messages/"+channel.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to get messages for channel %s: %w", channel.ID, err)
		}

		if messages == nil {
			continue // No messages for this channel
		}

		// Add the messages to the new spec
		bmPtr, err := readMsgpackSection[[]*BackupMessage](f, "messages/"+channel.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to get section: %w", err)
		}

		bm := *bmPtr

		if len(bm) == 0 {
			continue // No messages for this channel
		}

		var messagesList []discordgo.Message = make([]discordgo.Message, 0, len(bm))
		for _, msg := range bm {
			if msg.Message == nil {
				continue // Skip nil messages
			}
			msg := *msg.Message
			msg.Attachments = nil // Remove attachments as they are not needed in the new spec
			messagesList = append(messagesList, msg)
		}

		if len(messagesList) == 0 {
			continue // No valid messages for this channel
		}

		channelAllocations[channel.ID] = len(messagesList)
		messagesMap[channel.ID] = messagesList
	}

	var coreBackupData = CoreBackupData{
		Guild:             *srcGuild,
		Channels:          channelsList,
		Messages:          messagesMap,
		Options:           newBo,
		ChannelAllocation: channelAllocations,
	}

	var guildIcon bool
	var guildBanner bool
	var guildSplash bool
	for _, asset := range newBo.BackupGuildAssets {
		// Note: newBo is new spec so we use icon/banner/splash instead of guildIcon/guildBanner/guildSplash
		switch asset {
		case "icon":
			guildIcon = true
		case "banner":
			guildBanner = true
		case "splash":
			guildSplash = true
		default:
			return nil, fmt.Errorf("unknown guild asset: %s", asset)
		}
	}

	var tarfile = NewTarFile()

	// 4. guild icon, banner, splash
	addAsset := func(oldAssetPath string, newAssetPath string) error {
		bytes, err := f.Get(oldAssetPath)

		if err != nil {
			return fmt.Errorf("failed to get guild %s: %w", oldAssetPath, err)
		}

		if bytes == nil || bytes.Len() == 0 {
			return fmt.Errorf("guild asset %s is empty, likely an internal error", oldAssetPath)
		}

		err = tarfile.WriteSection(bytes, newAssetPath)

		if err != nil {
			return fmt.Errorf("failed to write guild %s: %w", oldAssetPath, err)
		}

		return nil
	}

	if guildIcon {
		err = addAsset("assets/guildIcon", "assets/icon.jpg")
		if err != nil {
			return nil, fmt.Errorf("failed to add guild icon: %w", err)
		}
	}

	if guildBanner {
		err = addAsset("assets/guildBanner", "assets/banner.jpg")
		if err != nil {
			return nil, fmt.Errorf("failed to add guild banner: %w", err)
		}
	}

	if guildSplash {
		err = addAsset("assets/guildSplash", "assets/splash.jpg")
		if err != nil {
			return nil, fmt.Errorf("failed to add guild splash: %w", err)
		}
	}

	// Write guild data
	err = tarfile.WriteJsonGzSection(coreBackupData, "core.json.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to write core backup data: %w", err)
	}

	databytes, err := tarfile.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build tar file: %w", err)
	}

	return databytes.Bytes(), nil
}
