package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetUnmarshalJSON(t *testing.T) {
	assetBytes := `{
		"type": "audience",
		"description": "just a desc",
		"data": {
		  "gender": "male",
		  "birthCountry": "Greece",
		  "socialMediaHoursUsage": 0,
		  "ageGroup": {
		    "min": 15,
		    "max": 30
		  }
		}
	      }`

	var a asset

	a.UnmarshalJSON([]byte(assetBytes))

	assert.EqualValues(t, "audience", a.Type)
	assert.Equal(t, "just a desc", a.Desc)
	au := a.Data.(*audience)
	assert.EqualValues(t, male, au.Gender)
	assert.Equal(t, "Greece", au.BirthCountry)
	assert.EqualValues(t, 0, au.SocialMediaHoursUsage)
	assert.EqualValues(t, 15, au.AgeGroup.Min)
	assert.EqualValues(t, 30, au.AgeGroup.Max)
}
