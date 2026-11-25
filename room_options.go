package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func resolveRoomUUID(c *NomiClient, selector string) (string, error) {
	rooms, err := c.GetRooms()
	if err != nil {
		return "", fmt.Errorf("listing rooms: %w", err)
	}
	var uuid string
	matches := 0
	for _, room := range rooms {
		if strings.EqualFold(room.UUID, selector) {
			return room.UUID, nil
		}
		if strings.EqualFold(room.Name, selector) {
			uuid = room.UUID
			matches++
		}
	}
	switch matches {
	case 0:
		return "", fmt.Errorf("no room matches %q. Run list-rooms to see available rooms", selector)
	case 1:
		return uuid, nil
	default:
		return "", fmt.Errorf("multiple rooms named %q. Use a UUID from list-rooms", selector)
	}
}

func validateRoomName(name string) error {
	if utf8.RuneCountInString(name) > 100 {
		return fmt.Errorf("room name must contain at most 100 characters")
	}
	return nil
}

func validateRoomNote(note string) error {
	if utf8.RuneCountInString(note) > 1000 {
		return fmt.Errorf("room note must contain at most 1000 characters")
	}
	return nil
}

func resolveRoomParticipants(c *NomiClient, participants []string) ([]string, error) {
	if len(participants) < 1 || len(participants) > 10 {
		return nil, fmt.Errorf("provide 1 to 10 participants using --nomi")
	}
	nomis, err := c.GetNomis()
	if err != nil {
		return nil, fmt.Errorf("listing Nomis to resolve room participants: %w", err)
	}
	uuids := make([]string, 0, len(participants))
	for _, participant := range participants {
		uuid, err := resolveRoomNomiUUID(nomis, participant)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

func resolveRoomNomiUUID(nomis []Nomi, participant string) (string, error) {
	var uuid string
	matches := 0
	for _, nomi := range nomis {
		if strings.EqualFold(nomi.UUID, participant) {
			return nomi.UUID, nil
		}
		if strings.EqualFold(nomi.Name, participant) {
			uuid = nomi.UUID
			matches++
		}
	}
	switch matches {
	case 0:
		return "", fmt.Errorf("no Nomi matches %q. Run list-nomis to see available names", participant)
	case 1:
		return uuid, nil
	default:
		return "", fmt.Errorf("multiple Nomis named %q. Use a UUID from list-nomis --full", participant)
	}
}
