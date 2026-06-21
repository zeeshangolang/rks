package main

import (
	"errors"
	"fmt"
	"net/http"

	"zeeshan.kineta.site/internal/model"
	"zeeshan.kineta.site/internal/model/validator"
)

func (app *Application) ActivateAccoun(w http.ResponseWriter, r *http.Request) {

	data := app.NewTemplateData(r)
	data.Form = EMailpasscode{}
	app.Render(w, 200, "confirmemail.html", data)

}

type EMailpasscode struct {
	Code string `form:"code"`

	validator.Validator `form:"-"`
}

//func(app *Application)

func (app *Application) ConfirmEmailPost(w http.ResponseWriter, r *http.Request) {

	var form EMailpasscode

	err := app.NewFormDecoder(r, &form)
	if err != nil {
		app.ServerError(w, err)
	}

	form.CheckField(validator.Notblank(form.Code), "code", "code cant be empty")
	form.CheckField(validator.MinChar(form.Code, 4), "code", "code should be minimum of 4 char")
	form.CheckField(validator.MaxChar(form.Code, 10), "code", "code cant be longer 8 char ")

	if !form.Valid() {
		data := app.NewTemplateData(r)
		data.Form = form
		app.Render(w, http.StatusBadRequest, "confirmemail.html", data)
		return
	}
	useridd := app.sessionmanager.GetInt(r.Context(), "userid")
	userid, err := app.Tokens.Confirmtoken(useridd, form.Code)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidCredentials):
			form.AddFieldError("code", "passcode is wrong or expired")
			data := app.NewTemplateData(r)
			data.Form = form
			app.Render(w, 200, "confirmemail.html", data)
			fmt.Printf("error %d", useridd)
		default:
			app.ServerError(w, err)
		}
		return
	}

	user, err := app.UserModel.UpdateUser(userid)
	if err != nil {
		app.ServerError(w, err)
	}
	data := app.NewTemplateData(r)
	data.User = user

	err = app.sessionmanager.RenewToken(r.Context())
	if err != nil {
		app.ServerError(w, err)
		return
	}
	IsActivated := true
	app.sessionmanager.Put(r.Context(), "authenticateduserid", userid)
	app.sessionmanager.Put(r.Context(), "isActivated", &IsActivated)
	http.Redirect(w, r, "/create", http.StatusSeeOther)
}
