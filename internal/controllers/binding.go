package controllers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// fieldNames are the request fields as the user knows them.
var fieldNames = map[string]string{
	"Email":           "l'adresse email",
	"Password":        "le mot de passe",
	"NewPassword":     "le nouveau mot de passe",
	"ConfirmPassword": "la confirmation du mot de passe",
	"CurrentPassword": "le mot de passe actuel",
	"Token":           "le lien",
}

// bindingMessage turns a request validation failure into a sentence for the
// user, instead of the validator's internal wording.
func bindingMessage(err error) string {
	var invalid validator.ValidationErrors
	if !errors.As(err, &invalid) || len(invalid) == 0 {
		return "Requête invalide."
	}
	f := invalid[0]
	name, ok := fieldNames[f.Field()]
	if !ok {
		name = f.Field()
	}
	switch f.Tag() {
	case "required":
		return "Merci de renseigner " + name + "."
	case "email":
		return "L'adresse email n'est pas valide."
	case "min":
		return "Le mot de passe doit faire au moins " + f.Param() + " caractères."
	case "max":
		return capitalize(name) + " est trop long."
	}
	return capitalize(name) + " n'est pas valide."
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}

// internalError logs what went wrong and tells the client only that it did:
// database or network details are not for the outside.
func internalError(c *gin.Context, err error) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne, réessayez dans un instant."})
}
