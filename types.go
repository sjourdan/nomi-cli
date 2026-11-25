package main

type Nomi struct {
	UUID             string `json:"uuid"`
	Gender           string `json:"gender"`
	Name             string `json:"name"`
	Created          string `json:"created"`
	RelationshipType string `json:"relationshipType"`
}

type NomiResponse struct {
	Nomis []Nomi `json:"nomis"`
}

type Room struct {
	UUID                  string `json:"uuid"`
	Name                  string `json:"name"`
	Created               string `json:"created"`
	Updated               string `json:"updated"`
	Status                string `json:"status"`
	BackchannelingEnabled bool   `json:"backchannelingEnabled"`
	Nomis                 []Nomi `json:"nomis"`
	Note                  string `json:"note"`
}

// RoomResponse represents the API response for listing rooms
type RoomResponse struct {
	Rooms []Room `json:"rooms"`
}

type CreateRoomRequest struct {
	Name                  string   `json:"name"`
	Note                  string   `json:"note"`
	BackchannelingEnabled bool     `json:"backchannelingEnabled"`
	NomiUUIDs             []string `json:"nomiUuids"`
}

type UpdateRoomRequest struct {
	Name                  *string  `json:"name,omitempty"`
	Note                  *string  `json:"note,omitempty"`
	BackchannelingEnabled *bool    `json:"backchannelingEnabled,omitempty"`
	NomiUUIDs             []string `json:"nomiUuids,omitempty"`
}
