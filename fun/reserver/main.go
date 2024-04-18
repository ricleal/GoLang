package main

import (
	"fmt"

	"github.com/google/uuid"
)

func UUIDV5(s string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(s))
}

// Profile
type Profile struct {
	id   uuid.UUID
	spec string
}

func NewProfile(spec string) *Profile {
	return &Profile{
		id:   UUIDV5(spec),
		spec: spec,
	}
}

// Reserver
type Reserver struct {
	profiles []*Profile
	recent   map[uuid.UUID]*Profile
}

func NewReserver() *Reserver {
	return &Reserver{make([]*Profile, 0), make(map[uuid.UUID]*Profile)}
}

func (r *Reserver) AddProfile(p *Profile) {
	r.profiles = append(r.profiles, p)
	r.recent[p.id] = p
}

func (r *Reserver) PopProfile() *Profile {
	if len(r.profiles) == 0 {
		return nil
	}
	p := r.profiles[0]
	r.profiles = r.profiles[1:]
	return r.recent[p.id]
}

func main() {
	r := NewReserver()
	n := 10
	for i := 0; i < n; i++ {
		r.AddProfile(NewProfile(fmt.Sprintf("%d", i%3)))
	}
	// print the profiles
	for p := r.PopProfile(); p != nil; p = r.PopProfile() {
		fmt.Printf("Profile ID: %s, Spec: %s\n", p.id, p.spec)
	}
}
