package handler

import (
	"fmt"
	"regexp"
        "time"
        "strings"

	"github.com/ameliagapin/reservebot/data"
	e "github.com/ameliagapin/reservebot/err"
	"github.com/ameliagapin/reservebot/models"
	"github.com/ameliagapin/reservebot/util"
	log "github.com/sirupsen/logrus"
)

var (
	actions = map[string]regexp.Regexp{
		"hello":         *regexp.MustCompile(`hello.+`),
		"reserve":       *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sreserve\s(\S+)\s?(.+)?`),
		"release":       *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\srelease\s(.+)`),
		"clear":         *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sclear\s(.+)`),
		"kick":          *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\skick\s\<\@([a-zA-Z0-9]+)\>`),
		"remove":        *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sremove\sme\sfrom\s(.+)`),
		"all_status":    *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sstatus$`),
		"single_status": *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sstatus\s(.+)`),
		"my_status":     *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\smy\sstatus`),
		"nuke":          *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\snuke$`),
		"prune":         *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\sprune$`),
		"help":          *regexp.MustCompile(`(?m)^\<\@[A-Z0-9]+\>\shelp$`),

		"reserve_dm":       *regexp.MustCompile(`(?m)^reserve\s(\S+)\s?(.+)?`),
		"release_dm":       *regexp.MustCompile(`(?m)^release\s(.+)`),
		"clear_dm":         *regexp.MustCompile(`(?m)^clear\s(.+)`),
		"kick_dm":          *regexp.MustCompile(`(?m)^kick\s\<\@([a-zA-Z0-9]+)\>`),
		"remove_dm":        *regexp.MustCompile(`(?m)^remove\sme\sfrom\s(.+)`),
		"all_status_dm":    *regexp.MustCompile(`(?m)^status$`),
		"single_status_dm": *regexp.MustCompile(`(?m)^status\s(.+)`),
		"my_status_dm":     *regexp.MustCompile(`(?m)^my\sstatus`),
		"nuke_dm":          *regexp.MustCompile(`(?m)^nuke$`),
		"prune_dm":         *regexp.MustCompile(`(?m)^prune$`),
		"help_dm":          *regexp.MustCompile(`(?m)^help$`),
	}
)

var (
        msgAlreadyInAllQueues           = "You are already in all specified queues"
        msgIDontKnow                    = "I don't know what happened, but it wasn't good"
        msgMustSpecifyResource          = "you must specify a resource"
        msgMustSpecifyUser              = "you must specify a user to kick"
        msgMustSpecifyValidResource     = "you must specify a valid resource"
        msgMustUseReleaseForY           = "you cannot remove yourself from the queue for `%s` because you currently have it. Please use `release` instead."
        msgMustUseRemoveForY            = "you cannot release `%s` because you do not currently have it. Please use `remove` instead."
        msgNoReservations               = "There are currently no reservations."
        msgPeriodItIsNowFree            = ". It is now free."
        msgPeriodXHasItCurrently        = ". %s has it currently."
        msgPeriodXStillHasIt            = ". %s still has it."
        msgQueuesPruned                 = "I have removed all unreserved resources."
        msgReservedButNotInQueue        = "%s reserved `%s`, but is currently not in the queue"
        msgResourceDoesNotExistY        = "resource `%s` does not exist"
        msgResourceImproperlyFormatted  = "Resources must be formatted as `<env>|<name>`. Example: `your_family|mom`"
        msgUknownUser                   = "I'm sorry, I don't know who that is."
        msgXClearedY                    = "%s cleared `%s`"
        msgXCurrentlyHas                = "%s currently has `%s`"
        msgXHasBeenKickedFromNResources = "%s has been kicked from %d resouce(s)"
        msgXHasBeenRemovedFromY         = "%s has been kicked from `%s`."
        msgXHasBeenRemovedFromYZ        = "%s has been removed from the queue for `%s`%s"
        msgXHasReleasedYItIsYours       = "%s has released `%s`."
        msgXHasReleasedYZ               = "%s has released `%s`. %s"
        msgXHasRemovedThemselvesFromYZ  = "%s has removed themselves from the queue for `%s`%s"
        msgXItIsYours                   = "%s it's all yours."
        msgXKickedYouFromY              = "%s kicked you from `%s`"
        msgXNukedQueue                  = "%s nuked all reservations."
        msgYHasBeenCleared              = "`%s` has been cleared"
        msgYouAreNInLineForY            = "you are %s in line for `%s`%s"
        msgYouAreNotInLineForY          = "you are not in line for `%s`"
        msgYouCurrentlyHave             = "you currently have `%s`"
        msgYouHaveNoReservations        = "you have no reservations"
        msgYouHaveReleasedY             = "you have released `%s`"
        msgYouHaveRemovedXFromY         = "you have removed %s from `%s`"
        msgYouHaveRemovedYourselfFromY  = "you have removed yourself from `%s`"
        msgExpirySoon                   = "WARNING: Your reservation of `%s` will expire in one minute. Please re-reserve the resource if you wish to extend your reservation."
)

func (h *Handler) getAction(text string) string {
	for a, r := range actions {
		if r.MatchString(text) {
			return a
		}
	}
	return ""
}

func (h *Handler) reserve(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	matches := h.getMatches(ea.Action, ev.Text)
	resources, err := h.getResourcesFromCommaList(matches[0])
	if err != nil {
		h.handleGetResourceError(ea, err)
		return err
	}

        durationStr := strings.Trim(matches[1], " ")

        var duration time.Duration
        duration, err = time.ParseDuration(durationStr)
        if err != nil {
            duration = time.Hour
        }

	success := []*models.Resource{}
	for _, res := range resources {
		err := h.data.Reserve(u, res.Name, res.Env, duration)
		if err != nil {
			// if the user is already in the queue, we're going to skip returning an error
			if err != e.AlreadyInQueue {
				h.errorReply(ev.Channel, err.Error())
				continue
			}
		}
		success = append(success, res)
	}

	if len(success) == 0 {
		return h.reply(ea, msgAlreadyInAllQueues, true)
	}

	for _, res := range success {
		pos, err := h.data.GetPosition(u, res.Name, res.Env)
		if err != nil {
			// This case really should never happen here, as we are only looping through our success cases
			log.Errorf("%+v", err)
			h.errorReply(ev.Channel, msgIDontKnow)
			return err
		}
		cu, err := h.data.GetReservationForResource(res.Name, res.Env)
		if err != nil {
			h.errorReply(ev.Channel, err.Error())
			log.Errorf("%+v", err)
			continue
		}
		switch pos {
		case 0:
			log.Errorf(msgReservedButNotInQueue, h.getUserDisplay(u, false), res)
		case 1:
			msg := fmt.Sprintf(msgYouCurrentlyHave, res)
			if ev.ChannelType != "im" {
				msg = fmt.Sprintf(msgXCurrentlyHas, h.getUserDisplayWithDuration(cu, true), res)
			}
			err = h.reply(ea, msg, false)
			if err != nil {
				log.Errorf("%+v", err)
			}
		default:
			c := ""
			if cu != nil {
				c = fmt.Sprintf(msgPeriodXHasItCurrently, h.getUserDisplayWithDuration(cu, false))
			}
			msg := fmt.Sprintf(msgYouAreNInLineForY, util.Ordinalize(pos), res, c)
			err = h.reply(ea, msg, true)
			if err != nil {
				log.Errorf("%+v", err)
			}
		}
	}

	return nil
}

func (h *Handler) release(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	matches := h.getMatches(ea.Action, ev.Text)
	resources, err := h.getResourcesFromCommaList(matches[0])
	if err != nil {
		h.handleGetResourceError(ea, err)
		return err
	}

	success := []*models.Resource{}
	for _, res := range resources {
		r := h.data.GetResource(res.Name, res.Env, false)
		if r == nil {
			h.errorReply(ev.Channel, fmt.Sprintf(msgResourceDoesNotExistY, res))
			continue
		}

		pos, err := h.data.GetPosition(u, res.Name, res.Env)
		if err != nil {
			if err == e.NotInQueue {
				h.errorReply(ev.Channel, fmt.Sprintf(msgYouAreNotInLineForY, res))
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		switch pos {
		case 0:
			h.errorReply(ev.Channel, fmt.Sprintf(msgYouAreNotInLineForY, res))
			continue
		case 1:
			err := h.data.Remove(u, res.Name, res.Env)
			if err != nil {
				if err == e.NotInQueue {
					h.reply(ea, fmt.Sprintf(msgMustUseRemoveForY, res), true)
					continue
				}
				h.errorReply(ev.Channel, err.Error())
				continue
			}
			success = append(success, res)
		default:
			h.reply(ea, fmt.Sprintf(msgMustUseRemoveForY, res), true)
			continue
		}
	}

	for _, res := range success {
		cu, err := h.data.GetReservationForResource(res.Name, res.Env)
		if err != nil {
			if err == e.ResourceDoesNotExist {
				h.errorReply(ev.Channel, fmt.Sprintf(msgResourceDoesNotExistY, res))
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		if ea.Event.ChannelType == "im" {
			// Confirm for user
			msg := fmt.Sprintf(msgYouHaveReleasedY, res)
			h.reply(ea, msg, false)

			if cu != nil {
				// Let next user know they are up
				msg = fmt.Sprintf(msgXHasReleasedYItIsYours, h.getUserDisplay(u, false), res)
				h.announce(ea, cu.User, msg)
			}
		} else {
			msg := msgPeriodItIsNowFree
			if cu != nil {
				msg = fmt.Sprintf(msgXItIsYours, h.getUserDisplay(cu.User, true))
			}
			msg = fmt.Sprintf(msgXHasReleasedYZ, h.getUserDisplay(u, false), res, msg)
			h.reply(ea, msg, false)
		}
	}

	return nil
}

func (h *Handler) remove(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	matches := h.getMatches(ea.Action, ev.Text)
	resources, err := h.getResourcesFromCommaList(matches[0])
	if err != nil {
		h.handleGetResourceError(ea, err)
		return err
	}

	for _, res := range resources {
		r := h.data.GetResource(res.Name, res.Env, false)
		if r == nil {
			h.errorReply(ev.Channel, fmt.Sprintf(msgResourceDoesNotExistY, res))
			continue
		}

		pos, err := h.data.GetPosition(u, res.Name, res.Env)
		if err != nil {
			if err == e.NotInQueue {
				h.errorReply(ev.Channel, fmt.Sprintf(msgYouAreNotInLineForY, res))
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		switch pos {
		case 0:
			h.errorReply(ev.Channel, fmt.Sprintf(msgYouAreNotInLineForY, res))
			continue
		case 1:
			h.reply(ea, fmt.Sprintf(msgMustUseReleaseForY, res), true)
			continue
		default:
			err = h.data.Remove(u, res.Name, res.Env)
			if err != nil {
				h.errorReply(ev.Channel, err.Error())
				continue
			}

			cu, err := h.data.GetReservationForResource(res.Name, res.Env)
			if err != nil {
				h.errorReply(ev.Channel, err.Error())
				continue
			}

			if ev.ChannelType == "im" {
				// We will need to confirm to the user
				h.reply(ea, fmt.Sprintf(msgYouHaveRemovedYourselfFromY, res), false)
			} else {
				// We only need to send one message in channel
				current := msgPeriodItIsNowFree
				if cu != nil {
					current = fmt.Sprintf(msgPeriodXStillHasIt, h.getUserDisplayWithDuration(cu, false))
				}
				msg := fmt.Sprintf(msgXHasRemovedThemselvesFromYZ, h.getUserDisplay(u, true), res, current)
				h.reply(ea, msg, false)
			}
		}
	}

	return nil
}

func (h *Handler) allStatus(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	userOnly := false
	switch ea.Action {
	case "my_status", "my_status_dm":
		userOnly = true
	}

	all := h.data.GetResources()

	if len(all) == 0 {
		return h.reply(ea, msgNoReservations, false)
	}

	resp := ""
	for _, res := range all {
		if userOnly {
			// Discarding the err here. Func returns 0 when there's an err so we'll use that as an indication
			// to just skip
			pos, _ := h.data.GetPosition(u, res.Name, res.Env)
			if pos <= 0 {
				continue
			}
		}
		msg, err := h.getCurrentResText(res, false)
		if err != nil {
			log.Errorf("%+v", err)
			h.errorReply(ev.Channel, "")
			continue
		}

		resp += msg + "\n"
	}

	if resp == "" {
		if userOnly {
			resp = msgYouHaveNoReservations
		} else {
			resp = msgNoReservations
		}
	}
	// Only address the user if they asked for *their* status
	h.reply(ea, resp, userOnly)

	return nil
}

func (h *Handler) singleStatus(ea *EventAction) error {
	ev := ea.Event
	r := h.getMatches(ea.Action, ev.Text)

	if len(r) == 0 {
		h.errorReply(ev.Channel, msgMustSpecifyResource)
		return nil
	}

	res, err := h.parseResource(r[0])
	if err != nil {
		// Probably don't need to insult the user for resource formatting here
		h.errorReply(ev.Channel, msgMustSpecifyValidResource)
		return nil
	}

	msg, err := h.getCurrentResText(res, false)
	if err != nil {
		if err == e.ResourceDoesNotExist {
			h.errorReply(ev.Channel, fmt.Sprintf(msgResourceDoesNotExistY, res))
			return nil
		}
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	h.reply(ea, msg, false)

	return nil
}

func (h *Handler) clear(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	matches := h.getMatches(ea.Action, ev.Text)
	resources, err := h.getResourcesFromCommaList(matches[0])
	if err != nil {
		h.handleGetResourceError(ea, err)
		return err
	}

	for _, res := range resources {
		q, err := h.data.GetQueueForResource(res.Name, res.Env)
		if err != nil {
			if err == e.ResourceDoesNotExist {
				h.errorReply(ev.Channel, fmt.Sprintf(msgResourceDoesNotExistY, res))
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		err = h.data.ClearQueueForResource(res.Name, res.Env)
		if err != nil {
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		msg := fmt.Sprintf(msgYHasBeenCleared, res)
		h.reply(ea, msg, false)

		// If request was via IM, we need to notify other users
		if ev.ChannelType == "im" {
			for _, r := range q.Reservations {
				if r.User.ID != ev.User {
					h.announce(ea, r.User, fmt.Sprintf(msgXClearedY, h.getUserDisplay(u, true), res))
				}
			}
		}
	}

	return nil
}

func (h *Handler) kick(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	matches := h.getMatches(ea.Action, ev.Text)
	if len(matches) != 1 {
		h.reply(ea, msgMustSpecifyUser, true)
		return nil
	}
	uToKick, err := h.getUser(matches[0])
	if err != nil {
		log.Errorf("%+v", err)
		h.reply(ea, msgUknownUser, true)
		return err
	}

	count := 0
	for _, res := range h.data.GetResources() {
		pos, err := h.data.GetPosition(uToKick, res.Name, res.Env)
		if err != nil {
			if err == e.NotInQueue {
				// this error does not need to be reported to the user
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}
		if pos != 1 {
			continue
		}

		err = h.data.Remove(uToKick, res.Name, res.Env)
		if err != nil {
			if err == e.NotInQueue {
				// this error does not need to be reported to the user
				continue
			}
			h.errorReply(ev.Channel, err.Error())
			continue
		}
		count++

		cu, err := h.data.GetReservationForResource(res.Name, res.Env)
		if err != nil {
			h.errorReply(ev.Channel, err.Error())
			continue
		}

		if ev.ChannelType == "im" {
			// We will need to confirm to the user
			h.reply(ea, fmt.Sprintf(msgYouHaveRemovedXFromY, h.getUserDisplay(uToKick, true), res), false)

			// If someone now has the resource, we must alert them
			if cu != nil {
				msg := fmt.Sprintf(msgXHasBeenRemovedFromY, h.getUserDisplay(uToKick, false), res)
				h.announce(ea, cu.User, msg)
			}

			// Alert user who was kicked
			msg := fmt.Sprintf(msgXKickedYouFromY, h.getUserDisplay(u, true), res)
			h.announce(ea, uToKick, msg)
		} else {
			// We only need to send one message in channel
			current := msgPeriodItIsNowFree
			if cu != nil {
				current = fmt.Sprintf(msgPeriodXHasItCurrently, h.getUserDisplayWithDuration(cu, false))
			}

			msg := fmt.Sprintf(msgXHasBeenRemovedFromYZ, h.getUserDisplay(u, false), res, current)
			h.reply(ea, msg, false)
		}
	}

	msg := fmt.Sprintf(msgXHasBeenKickedFromNResources, h.getUserDisplay(uToKick, true), count)
	h.reply(ea, msg, false)

	// User will need to be alerted
	return nil
}

func (h *Handler) nuke(ea *EventAction) error {
	ev := ea.Event
	u, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	h.data = data.NewMemory()

	msg := fmt.Sprintf(msgXNukedQueue, h.getUserDisplay(u, true))
	h.reply(ea, msg, false)

	return nil
}

func (h *Handler) prune(ea *EventAction) error {
	ev := ea.Event
	_, err := h.getUser(ev.User)
	if err != nil {
		log.Errorf("%+v", err)
		h.errorReply(ev.Channel, "")
		return err
	}

	resources := h.data.GetResources()
	for _, res := range resources {
		q, err := h.data.GetQueueForResource(res.Name, res.Env)
		if err != nil {
			// this shouldn't happen, but there's nothing to alert the user to
			log.Errorf("%+v", err)
			continue
		}

		if q.HasReservations() {
			continue
		}

		err = h.data.RemoveResource(res.Name, res.Env)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
	}

	h.reply(ea, msgQueuesPruned, false)

	return nil
}

func (h *Handler) Warn() error {

        checkInterval := time.Second * 60
        //timeLimit := time.Hour
        //timeLimit := time.Minute * 5

        for {
            time.Sleep(checkInterval)
            reservations := h.data.GetReservations()
            for _, res := range reservations {
                //if time.Since(res.Time) > timeLimit {
                if res.TimeRemaining().Seconds() <= 60 {
                    h.sendDM(res.User, fmt.Sprintf(msgExpirySoon, res.Resource))
                }
            }
        }
}
