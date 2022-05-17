package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type assetType string

const (
	chartType    assetType = "chart"
	insightType  assetType = "insight"
	audienceType assetType = "audience"
)

type gender string

const (
	male   gender = "male"
	female gender = "female"
)

type user struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (u *user) validate() error {
	if u.Email == "" {
		return errors.New("email missing")
	}
	if u.Name == "" {
		return errors.New("name missing")
	}
	return nil
}

// overwrite u's fields with the non nil fields of y.
func (u *user) update(y *user) {
	if y.Email != "" {
		u.Email = y.Email
	}
	if y.Name != "" {
		u.Name = y.Name
	}
}

type asset struct {
	ID   uint      `json:"id"`
	Type assetType `json:"type"`
	Desc string    `json:"description"`
	Data any       `json:"data"`
}

// ensure that a.Type is compatible with a.Data
func (a *asset) validate() error {
	var ok bool
	switch a.Type {
	case chartType:
		_, ok = a.Data.(*chart)
	case insightType:
		_, ok = a.Data.(*insight)
	case audienceType:
		_, ok = a.Data.(*audience)
	}
	if !ok {
		return errors.New("incompatible asset type and data")
	}
	return nil
}

// custom unmarshal. Delay the decoding of Data field until its type is parsed.
func (a *asset) UnmarshalJSON(b []byte) error {
	type Alias struct {
		ID   uint            `json:"id"`
		Type assetType       `json:"type"`
		Desc string          `json:"description"`
		Data json.RawMessage `json:"data"`
	}

	var alias Alias
	err := json.Unmarshal(b, &alias)
	if err != nil {
		return err
	}

	a.ID = alias.ID
	a.Type = alias.Type
	a.Desc = alias.Desc
	switch a.Type {
	case chartType:
		var ch *chart
		err := json.Unmarshal(alias.Data, &ch)
		if err != nil {
			return err
		}
		a.Data = ch
	case insightType:
		var in *insight
		err := json.Unmarshal(alias.Data, &in)
		if err != nil {
			return err
		}
		a.Data = in
	case audienceType:
		var au *audience
		err := json.Unmarshal(alias.Data, &au)
		if err != nil {
			return err
		}
		switch au.Gender {
		case male, female, "":
		default:
			return fmt.Errorf("unknown gender %s", au.Gender)
		}
		a.Data = au
	default:
		return fmt.Errorf("unknown asset type: %s", a.Type)
	}
	return nil

}

// overwrite a's fields with the non nil fields of b.
func (a *asset) update(b *asset) {
	if b.Desc != "" {
		a.Desc = b.Desc
	}
	if b.Data != nil {
		switch v := b.Data.(type) {
		case chart:
			v2, ok := a.Data.(*chart)
			if !ok {
				panic(v2)
			}
			if v.Title != "" {
				v2.Title = v.Title
			}
			if v.TitleAxisX != "" {
				v2.TitleAxisX = v.TitleAxisX
			}
			if v.TitleAxisY != "" {
				v2.TitleAxisY = v.TitleAxisY
			}
			if v.Data != nil {
				v2.Data = v.Data
			}
			a.Data = v2
			a.Type = chartType
		case insight:
			v2, ok := a.Data.(*insight)
			if !ok {
				panic(v2)
			}
			if v.Text != "" {
				v2.Text = v.Text
			}
			a.Data = v2
			a.Type = insightType
		case audience:
			v2, ok := a.Data.(*audience)
			if !ok {
				panic(v2)
			}
			if v.Gender != "" {
				v2.Gender = v.Gender
			}
			if v.AgeGroup.Max != 0 {
				v2.AgeGroup.Max = v.AgeGroup.Max

			}
			if v.AgeGroup.Min != 0 {
				v2.AgeGroup.Min = v.AgeGroup.Min
			}
			if v.SocialMediaHoursUsage != 0 {
				v2.SocialMediaHoursUsage = v.SocialMediaHoursUsage
			}
			if v.BirthCountry != "" {
				v2.BirthCountry = v.BirthCountry
			}
			a.Data = v2
			a.Type = audienceType
		default:
			panic(v)
		}
	}
}

type chart struct {
	Title      string `json:"title"`
	TitleAxisX string `json:"titleAxisX"`
	TitleAxisY string `json:"titleAxisY"`
	Data       []byte `json:"data"`
}

type insight struct {
	Text string `json:"text"`
}

type audience struct {
	Gender                gender   `json:"gender"`
	BirthCountry          string   `json:"birthCountry"`
	SocialMediaHoursUsage uint8    `json:"socialMediaHoursUsage"`
	AgeGroup              ageGroup `json:"ageGroup"`
}

func (a *audience) String() string {
	return fmt.Sprintf("%ss born in %s in the age group of %s spent %d hours on social media",
		strings.Title(string(a.Gender)), a.BirthCountry, a.AgeGroup, a.SocialMediaHoursUsage,
	)
}

// Custom marshaler.
// Adds a string field which describes the audience object in a human readable text.
func (a *audience) MarshalJSON() ([]byte, error) {
	type Alias audience
	var alias Alias = Alias(*a)
	return json.Marshal(&struct {
		Str string `json:"string"`
		*Alias
	}{
		Str:   a.String(),
		Alias: &alias,
	})

}

type ageGroup struct {
	Min uint8 `json:"min"`
	Max uint8 `json:"max"`
}

func (a ageGroup) String() string {
	return fmt.Sprintf("%d-%d", a.Min, a.Max)
}
