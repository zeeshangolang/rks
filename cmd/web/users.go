package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"zeeshan.kineta.site/internal/model"
	"zeeshan.kineta.site/internal/model/validator"
)

func (app *Application) Signup(w http.ResponseWriter, r *http.Request) {
	var Signup Signup
	data := app.NewTemplateData(r)
	data.Form = Signup
	app.Render(w, 200, "signup.html", data)
}

type Signup struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	ProfileImage        string `form:"profileimg"`
	validator.Validator `form:"-"`
}

func (app *Application) SignupPost(w http.ResponseWriter, r *http.Request) {
	var form Signup
	app.NewFormDecoder(r, &form)
	//VALIDATING the form filled by users
	form.CheckField(validator.MatchUserName(validator.Usernamerx, form.Name), "name", "use Alphabets only")
	form.CheckField(validator.MinChar(form.Name, 3), "name", "name should be minimum 3 characters long")
	form.CheckField(validator.MaxChar(form.Name, 20), "name", "name cant be longer than 20 characters")
	form.CheckField(validator.MatchEmail(validator.EmailRx, form.Email), "email", "Invalid email")
	form.CheckField(validator.MaxChar(form.Email, 40), "email", "email cant be 40 char long")
	form.CheckField(validator.MinChar(form.Password, 8), "password", "password should be minimum of 8 Characters long")
	form.CheckField(validator.MaxChar(form.Password, 40), "password", "password cant be 40 cvhar long")
	// confirming if form is valid if not reshow the form with errors
	if !form.Valid() {
		data := app.NewTemplateData(r)
		data.Form = form
		data.ShowNav = true
		app.Render(w, 200, "signup.html", data)
		return
	}
	//INSERTING THE USER info to database also checking for error
	//if error of duplicate email show up tell the user email already in use
	userid, err := app.UserModel.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrDuplicateemail):
			form.AddFieldError("email", "email already in use ")
			data := app.NewTemplateData(r)
			data.Form = form
			app.Render(w, 200, "signup.html", data)

		default:
			app.ServerError(w, err)
		}
		return
	}
	//creating a random 6 digitas code for email verification
	plaintextpass, err := model.NewRandomINt()
	if err != nil {
		app.ServerError(w, err)
		return
	}
	expiry := time.Now().Add(3 * time.Hour)
	//insering that 6 digits random code to database
	err = app.Tokens.InsertToken(int32(userid), plaintextpass, expiry)
	if err != nil {
		app.Errorlog.Printf("%s", err)
		return
	}
	//creating the data for our our email temp that will be send to user
	data := app.NewEmailtempdata(r)
	data.Token = plaintextpass
	data.Name = form.Name
	// go routine to send the email
	go func() {
		err = app.Mailer.Send(form.Email, "usertemplate.html", data)
		if err != nil {
			app.Errorlog.Printf("%s", err.Error())

		}
	}()
	attempts, err := app.Tokens.UpdateTokenCount(userid)
	data.attempts = attempts
	app.sessionmanager.Put(r.Context(), "flash", "Signup successfull, Please login now")
	app.sessionmanager.Put(r.Context(), "userid", userid)
	http.Redirect(w, r, "/confirmemail", http.StatusSeeOther)
}

type Login struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *Application) login(w http.ResponseWriter, r *http.Request) {

	var form Login
	data := app.NewTemplateData(r)
	data.Form = form
	app.Render(w, 200, "login.html", data)
}

func (app *Application) LoginPost(w http.ResponseWriter, r *http.Request) {
	var form Login
	app.NewFormDecoder(r, &form)
	form.CheckField(validator.MatchEmail(validator.EmailRx, form.Email), "email", "must be a vlaid email ")
	form.CheckField(validator.MaxChar(form.Email, 40), "email", "email cant be longer ")
	form.CheckField(validator.Notblank(form.Password), "password", "password cant be blank")

	if !form.Valid() {
		data := app.NewTemplateData(r)
		data.Form = form
		app.Render(w, http.StatusBadRequest, "login.html", data)
		return
	}
	id, IsActivated, err := app.UserModel.Authenticate(form.Email, form.Password)
	if err != nil {

		if errors.Is(err, model.ErrInvalidCredentials) {
			form.Addnonfielderror("email or pass incorrect")
			data := app.NewTemplateData(r)
			data.Form = form
			app.Render(w, http.StatusBadRequest, "login.html", data)
			return
		}
		app.ServerError(w, err)

		return
	}
	err = app.sessionmanager.RenewToken(r.Context())
	if err != nil {
		fmt.Print("problem here")
	}
	app.sessionmanager.Put(r.Context(), "authenticateduserid", id)
	app.sessionmanager.Put(r.Context(), "isActivated", IsActivated)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Application) CheckingDbStats(w http.ResponseWriter, r *http.Request) {
	users, err := app.UserModel.Latest()
	if err != nil {
		app.ServerError(w, err)
		return
	}
	data := app.NewTemplateData(r)

	data.Users = users

	app.Render(w, 200, "usershtml.html", data)

}

func (app *Application) SearchUserProfile(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("q")

	// param := httprouter.ParamsFromContext(r.Context())

	// name := param.ByName("search")

	users, err := app.UserModel.GetUSERByName(query)
	fmt.Println(query)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrecordNotFound):
			app.NotFound(w)
		default:
			app.ServerError(w, err)
		}
		return
	}

	data := app.NewTemplateData(r)
	data.Users = users

	app.Render(w, 200, "usersearch.html", data)

}

func (app *Application) UserProfile(w http.ResponseWriter, r *http.Request) {

}
