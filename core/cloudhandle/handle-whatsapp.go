package cloudhandle

import (
	"context"
	"fmt"
	"time"

	mautrix "github.com/iKonoTelecomunicaciones/go"
	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/matrix/mxmain"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog/log"
)

// getDefaultPowerLevels retrieves the default power level settings from the connector's configuration.
func (whatsappConnector *WhatsappCloudConnector) getDefaultPowerLevels() (
	*bridgev2.PowerLevelOverrides, error,
) {
	levels := &bridgev2.PowerLevelOverrides{}
	default_power_levels := whatsappConnector.Config.DefaultPowerLevels

	if default_power_levels == nil {
		return nil, fmt.Errorf("default power levels are not configured")
	}

	levels.UsersDefault = default_power_levels.UsersDefault
	levels.EventsDefault = default_power_levels.EventsDefault
	levels.Invite = default_power_levels.Invite
	levels.Kick = default_power_levels.Kick
	levels.Ban = default_power_levels.Ban
	levels.Redact = default_power_levels.Redact

	return levels, nil
}

// getDefaultMembers creates a default list of members for a new portal,
// including the user and the bot, with their respective power levels.
func (whatsappConnector *WhatsappCloudConnector) getDefaultMembers(
	userLogin *bridgev2.UserLogin, userKey types.UserKey,
) map[networkid.UserID]bridgev2.ChatMember {
	userLoginPowerLevel := 100

	// Create a map of default members for the portal
	// This map will contain the user and the user login with their respective power levels.
	memberMap := map[networkid.UserID]bridgev2.ChatMember{
		networkid.UserID(userKey.MXID): {
			Membership: event.MembershipJoin,
			Nickname:   &userKey.Name,
			PowerLevel: &whatsappConnector.Config.DefaultUserLevel,
			UserInfo: &bridgev2.UserInfo{
				Name: &userKey.Name,
			},
		},
		networkid.UserID(userLogin.ID): {
			Membership: event.MembershipJoin,
			Nickname:   &userLogin.RemoteProfile.Name,
			PowerLevel: &userLoginPowerLevel,
			UserInfo: &bridgev2.UserInfo{
				Name: &userLogin.RemoteProfile.Name,
			},
		},
	}

	return memberMap
}

// inviteUsersToPortal invites the necessary users (the agent and the customer)
// to the portal's Matrix room.
func (whatsappConnector *WhatsappCloudConnector) inviteUsersToPortal(
	ctx context.Context,
	brmain mxmain.BridgeMain,
	portal *bridgev2.Portal,
	userLogin *bridgev2.UserLogin,
	customerMXID id.UserID,
) error {
	client := brmain.Matrix.Bot.Client
	// Invite the acd user to the portal's Matrix room
	_, err := client.InviteUser(
		ctx, portal.MXID, &mautrix.ReqInviteUser{UserID: id.UserID(userLogin.UserMXID)},
	)

	if err != nil {
		return err
	}

	// Invite the customer user to the portal's Matrix room
	_, err = brmain.Matrix.AS.Client(customerMXID).JoinRoom(
		ctx, string(portal.MXID), &mautrix.ReqJoinRoom{},
	)

	if err != nil {
		return err
	}

	return nil
}

// sendSetPlAndRelay sets the power levels for users in the Matrix room
// and sets up the relay mode for the portal.
func (whatsappConnector *WhatsappCloudConnector) sendSetPlAndRelay(
	ctx context.Context,
	portal *bridgev2.Portal,
	brmain mxmain.BridgeMain,
	userLogin *bridgev2.UserLogin,
	userID id.UserID,
) error {

	powerLevels, err := portal.Bridge.Matrix.GetPowerLevels(ctx, portal.MXID)

	if err != nil {
		return err
	}

	// Change the power level of the user
	powerLevels.Users[id.UserID(string(userLogin.UserMXID))] = 100
	powerLevels.Users[id.UserID(string(userID))] = whatsappConnector.Config.DefaultUserLevel

	botIntent := brmain.Bridge.Matrix.BotIntent()

	content := event.Content{
		Parsed: &powerLevels,
	}

	// Send the state event to the portal to set the power levels
	_, err = botIntent.SendState(
		ctx, portal.MXID, event.StatePowerLevels, "", &content, time.Now(),
	)

	if err != nil {
		return err
	}

	err = portal.SetRelay(ctx, userLogin)

	return err
}

// InitializatePortal orchestrates the initialization of a new portal
// by inviting users and setting power levels.
func (whatsappConnector *WhatsappCloudConnector) InitializatePortal(
	ctx context.Context,
	portal *bridgev2.Portal,
	userLogin *bridgev2.UserLogin,
	brmain mxmain.BridgeMain,
	userKey types.UserKey,
) error {
	log := whatsappConnector.Bridge.Log.With().Str("component", "HandleWhatsapp").Logger()

	err := whatsappConnector.inviteUsersToPortal(
		ctx, brmain, portal, userLogin, id.UserID(userKey.MXID),
	)

	if err != nil {
		log.Error().Err(err).Interface("Error", err).Msg("Failed to invite users to portal")
		return fmt.Errorf(
			"failed to invite users to portal: %w", err,
		)
	}
	log.Info().Msgf("Sending set power levels and relay for the portal %s", portal.MXID)
	err = whatsappConnector.sendSetPlAndRelay(
		ctx,
		portal,
		brmain,
		userLogin,
		id.UserID(userKey.MXID),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize portal")
		return fmt.Errorf("failed to send the set pl and relay: %w", err)
	}

	log.Info().Msg("Portal initialized successfully")
	return nil
}

// CreatePortalWithKey creates a new portal (and its corresponding Matrix room)
// using a specific portal key. It sets up the portal's name, members, and power levels.
func (whatsappConnector *WhatsappCloudConnector) CreatePortalWithKey(
	ctx context.Context,
	key networkid.PortalKey,
	userLogin *bridgev2.UserLogin,
	brmain mxmain.BridgeMain,
	userKey types.UserKey,
) (*bridgev2.Portal, error) {
	log := whatsappConnector.Bridge.Log.With().Str("component", "HandleWhatsapp").Logger()
	roomType := database.RoomTypeDM
	portal, err := whatsappConnector.Bridge.GetPortalByKey(ctx, key)

	if err != nil {
		whatsappConnector.Bridge.Log.Error().Err(err).Object("portal_key", key).
			Msg("Failed to get portal to handle remote event")
		return nil, fmt.Errorf("Failed to get portal with key %s: %w", key, err)
	}

	contactInfo := &types.ContactInfo{
		Found:     true,
		FirstName: userKey.Name,
		FullName:  userKey.Name,
	}

	portalName := whatsappConnector.Config.FormatDisplayname(
		userKey.Name, string(userKey.Name), *contactInfo,
	)

	info := &bridgev2.ChatInfo{
		Name: &portalName,
		Type: &roomType,
		JoinRule: &event.JoinRulesEventContent{
			JoinRule: event.JoinRulePublic,
		},
	}

	powerLevels, err := whatsappConnector.getDefaultPowerLevels()

	if err != nil {
		whatsappConnector.Bridge.Log.Error().Err(err).Object("portal_key", key).
			Msg("Failed to get power levels for portal")
		return nil, fmt.Errorf("Failed to get power levels for portal with key %s: %w", key, err)
	}

	defaultMembers := whatsappConnector.getDefaultMembers(userLogin, userKey)

	info.Members = &bridgev2.ChatMemberList{
		IsFull:         false,
		CheckAllLogins: false,
		PowerLevels:    powerLevels,
		MemberMap:      defaultMembers,
	}

	err = portal.CreateMatrixRoom(ctx, userLogin, info)

	if err != nil {
		whatsappConnector.Bridge.Log.Error().Err(err).Object("portal_key", key).
			Msg("Failed to create matrix room for portal")
		return nil, fmt.Errorf(
			"Failed to create matrix room for portal with key %s: %w", key, err,
		)
	}

	err = whatsappConnector.InitializatePortal(ctx, portal, userLogin, brmain, userKey)

	if err != nil {
		log.Error().Interface(
			"error", err,
		).Msg("Error while handling cloud event")

		return nil, fmt.Errorf("Error while initializing portal: %w", err)
	}

	return portal, nil
}

// GetPortalWithKey retrieves an existing portal using its key.
func (whatsappConnector *WhatsappCloudConnector) GetPortalWithKey(
	ctx context.Context, key networkid.PortalKey, userLogin *bridgev2.UserLogin,
) (portal *bridgev2.Portal, err error) {
	portal, err = whatsappConnector.Bridge.GetExistingPortalByKey(ctx, key)

	if err != nil {
		whatsappConnector.Bridge.Log.Error().Err(err).Object("portal_key", key).
			Msg("Failed to get portal to handle remote event")

		err = fmt.Errorf("failed to get portal with key %s: %w", key, err)
		return
	}

	return
}

// GetPortal is a wrapper that either gets an existing portal or creates a new one if it doesn't exist.
func (whatsappConnector *WhatsappCloudConnector) GetPortal(
	ctx context.Context,
	userLogin *bridgev2.UserLogin,
	brmain mxmain.BridgeMain,
	userKey types.UserKey,
) (*bridgev2.Portal, error) {
	portalKey := waid.MakePortalKey(string(userKey.ID))
	portal, err := whatsappConnector.GetPortalWithKey(ctx, portalKey, userLogin)

	if portal == nil {
		log.Info().Interface("portalID", portalKey.ID).Msg("Creating portal with key...")
		portal, err = whatsappConnector.CreatePortalWithKey(
			ctx, portalKey, userLogin, brmain, userKey,
		)
	}

	if err != nil {
		log.Error().Err(err).Msg("Error while getting portal with key: " + string(userKey.ID))
		return nil, fmt.Errorf("Error while getting portal with key %s: %w", string(userKey.ID), err)
	}

	if portal == nil {
		log.Error().Msg("Portal not found to handle remote event")
		return nil, fmt.Errorf("Portal not found to handle remote event with key %s", string(userKey.ID))
	}

	return portal, nil
}

// GetWhatsappCloudClient creates and returns a new WhatsappCloudClient instance for a given user login.
func (whatsappConnector *WhatsappCloudConnector) GetWhatsappCloudClient(
	ctx context.Context,
	userLogin *bridgev2.UserLogin,
) *WhatsappCloudClient {

	wClient := &WhatsappCloudClient{
		Main:      whatsappConnector,
		UserLogin: userLogin,
	}

	return wClient
}
