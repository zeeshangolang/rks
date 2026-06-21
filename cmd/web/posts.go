package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"zeeshan.kineta.site/internal/model"
	"zeeshan.kineta.site/internal/model/validator"
)

func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	app.Infolog.Print("handlers working")
	posts, err := app.Posts.Latest()
	if err != nil {
		app.ServerError(w, err)
		return
	}
	data := app.NewTemplateData(r)
	data.Posts = posts
	data.ShowSearcbar = true
	data.ShowNav = true
	app.Render(w, 200, "main.html", data)
}
func (app *Application) About(w http.ResponseWriter, r *http.Request) {
	data := app.NewTemplateData(r)
	app.Render(w, 200, "about.html", data)
}

func (app *Application) Premium(w http.ResponseWriter, r *http.Request) {
	data := app.NewTemplateData(r)
	app.Render(w, 200, "premium.html", data)
}
func (app *Application) View(w http.ResponseWriter, r *http.Request) {
	var form COmment
	var rateform RATING
	param := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.Atoi(param.ByName("id"))
	fmt.Printf("\n this is post id in View handler %d\n", id)
	if err != nil || id < 1 {
		app.NotFound(w)
		return
	}

	post, err := app.Posts.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrecordNotFound):
			app.NotFound(w)
		default:
			app.ServerError(w, err)
		}
		return
	}
	userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
	hasrated, err := app.Posts.CheckRated(userid, id)
	if err != nil {
		app.Errorlog.Print(err)
		return
	}

	number, err := app.Posts.ReturnDownloadCounts(userid)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrecordNotFound):

		default:
			fmt.Print(err)
			return
		}
	}
	excedded := number > 4

	data := app.NewTemplateData(r)

	data.Post = post
	data.HasRated = hasrated
	data.Form = form
	data.Rating = rateform
	data.IsOwner = (post.Userid == int32(userid))
	data.DCExcedded = excedded
	data.ShowNav = true
	app.Render(w, 200, "view.html", data)
}

type PostForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Category            string `form:"category"`
	ImageAddr           string `form:"Image"`
	PdfFile             string `form:"Pdf"`
	validator.Validator `form:"-"`
}

func (app *Application) PostForm(w http.ResponseWriter, r *http.Request) {

	var form PostForm

	data := app.NewTemplateData(r)
	data.Form = form
	data.ShowNav = true
	app.Render(w, 200, "create.html", data)

}

func (app *Application) PostPost(w http.ResponseWriter, r *http.Request) {
	var form PostForm
	r.ParseMultipartForm(10 << 20)
	app.NewFormDecoder(r, &form)

	form.CheckField(validator.Notblank(form.Title), "title", "THIS field must be filled ")
	form.CheckField(validator.Notblank(form.Content), "content", "THIS field must be filled ")
	form.CheckField(validator.Notblank(form.Category), "category", "THIS field must be filled ")
	form.CheckField(validator.MaxChar(form.Title, 100), "title", " MUST be less than 100 characters")
	form.CheckField(validator.MinChar(form.Title, 5), "title", "Must be larger than 5 keys")
	form.CheckField(validator.MinChar(form.Content, 5), "content", "Must be larger than 5 keys")
	form.CheckField(validator.MaxChar(form.Content, 1000), "content", " MUST be less than 1000 characters")

	if !form.Valid() {
		data := app.NewTemplateData(r)
		data.Form = form
		data.ShowNav = true
		app.Render(w, 201, "create.html", data)
		return
	}

	file, handler, err := r.FormFile("myfile")
	if err != nil {
		form.NonFieldError = append(form.NonFieldError, "File must be chosen")
		data := app.NewTemplateData(r)
		data.Form = form
		data.ShowNav = true
		app.Render(w, 201, "create.html", data)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d-%s",
		time.Now().Unix(),
		strings.ReplaceAll(handler.Filename, " ", ""),
	)

	go func() {
		app.UploadToSupabase(file, filename)
	}()

	pdfFile, pdfHandler, err := r.FormFile("Pdf")
	if err != nil {
		form.NonFieldError = append(form.NonFieldError, "Error on pdf")
		data := app.NewTemplateData(r)
		data.Form = form
		data.ShowSearcbar = false
		app.Render(w, 204, "create.html", data)
		return
	}
	pdffilename := fmt.Sprintf("%d-%s", time.Now().Unix(), pdfHandler.Filename)

	//pdfPublicUrl := fmt.Sprintf("https://f005.backblazeb2.com/file/zeeshanfirstbucket/%s", pdffilename)

	go func() {
		ctx := context.Background()
		app.CopyFile(ctx, app.Bucket, pdfFile, pdffilename)
		if err != nil {
			fmt.Print(err)
			return
		}
	}()

	publicURL := fmt.Sprintf(
		"https://eeutgcuqpkmykwdomhzl.supabase.co/storage/v1/object/public/zeeshan/%s",
		filename,
	)

	fileName := sql.NullString{String: publicURL, Valid: true}

	userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
	pid, err := app.Posts.Insert(
		form.Title,
		form.Content,
		form.Category,
		fileName,
		pdffilename,
		userid,
	)
	if err != nil {
		app.ServerError(w, err)
		return
	}
	fmt.Print(pid)

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", pid), http.StatusSeeOther)

}

type COmment struct {
	Comment string `form:"comment"`
}

type RATING struct {
	Rating              int `form:"Rating"`
	validator.Validator `form:"-"`
}

func (app *Application) Ratingpage(w http.ResponseWriter, r *http.Request) {

	app.Render(w, 200, "ratepostpage.html", nil)

}

func (app *Application) Rating(w http.ResponseWriter, r *http.Request) {
	var form RATING

	param := httprouter.ParamsFromContext(r.Context())
	postid, err := strconv.Atoi(param.ByName("id"))
	fmt.Printf("\n this is post id in Rating handler %d\n", postid)
	if err != nil || postid < 1 {
		app.ClientError(w, http.StatusBadRequest)
		return
	}

	app.NewFormDecoder(r, &form)
	form.CheckField(validator.PermittedValues(form.Rating, 1, 2, 3, 4, 5), "rating", "please choose rating")

	fmt.Printf("\n this is form.Rating Value %d\n", form.Rating)

	if !form.Valid() {
		app.ClientError(w, http.StatusBadRequest)
		return
	}

	userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")

	err = app.Posts.InsertRating(userid, postid, form.Rating)
	if err != nil {
		app.ServerError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postid), http.StatusSeeOther)

}

func (app *Application) MakeComment(w http.ResponseWriter, r *http.Request) {

	var form COmment
	err := app.NewFormDecoder(r, &form)
	if err != nil {
		app.Errorlog.Print(err)
		return
	}

	param := httprouter.ParamsFromContext(r.Context())
	postid, err := strconv.Atoi(param.ByName("id"))
	if err != nil || postid < 1 {
		app.ClientError(w, http.StatusBadRequest)
		return
	}
	userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")

	_, err = app.Posts.INsertComment(form.Comment, userid, postid)
	if err != nil {
		app.ServerError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postid), http.StatusSeeOther)

}

func (app *Application) PdfDownloader(w http.ResponseWriter, r *http.Request) {

	param := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(param.ByName("id"))

	if err != nil || id < 1 {
		app.ClientError(w, http.StatusBadRequest)
		return
	}

	pdfurl, err := app.Posts.GetPdfUrl(id)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrecordNotFound):
			app.NotFound(w)

		default:
			app.Errorlog.Print(err)
		}
		return
	}

	ctx := context.Background()

	token, err := app.Bucket.AuthToken(ctx, pdfurl, 1*time.Hour)

	downloadurl := fmt.Sprintf("https://f005.backblazeb2.com/file/zeeshanfirstbucket/%s?Authorization=%s",
		pdfurl,
		token)

	// data := app.NewTemplateData(r)
	// data.Post.PdfFile = downloadurl
	// app.Render(w, )
	uid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
	err = app.Posts.UpdateDownloadCounts(uid)
	fmt.Print(err)
	http.Redirect(w, r, downloadurl, http.StatusSeeOther)

}

func (app *Application) PostsByUser(w http.ResponseWriter, r *http.Request) {

	param := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(param.ByName("id"))
	if err != nil || id < 1 {
		app.ClientError(w, http.StatusBadRequest)
		return
	}

	posts, err := app.Posts.GetPostOfAUser(id)

	if err != nil {
		if errors.Is(err, model.ErrecordNotFound) {
			app.NotFound(w)
			return
		}
		app.ServerError(w, err)
		return
	}

	data := app.NewTemplateData(r)

	data.Posts = posts

	app.Render(w, 200, "userspost.html", data)

}

func (app *Application) SearchPost(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("sp")
	fmt.Print(query)
	posts, err := app.Posts.SearchPosts(query)

	if err != nil {
		switch {
		case errors.Is(err, model.ErrecordNotFound):
			fmt.Print("no results")
		default:
			fmt.Print(err)
		}

		return
	}

	data := app.NewTemplateData(r)
	data.Posts = posts
	data.ShowSearcbar = true

	app.Render(w, 200, "postssearch.html", data)

}

// //
// func (app *Application) buildViewData(r *http.Request, postID int) (*Template, error) {
// 	post, err := app.Posts.Get(postID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
// 	hasrated, _ := app.Posts.CheckRated(userid, postID)

// 	data := app.NewTemplateData(r)
// 	data.Post = post
// 	data.HasRated = hasrated
// 	data.Rating = RATING{} // default
// 	data.Form = COmment{}  // default

// 	return data, nil
// }
