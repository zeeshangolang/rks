package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/go-playground/form/v4"
)

func (app *Application) IsActivated(r *http.Request) bool {
	IsActivated, ok := r.Context().Value(IsActivatedContextKey).(bool)
	if !ok {
		return false
	}
	return IsActivated
}

func (app *Application) IsAuthenticated(r *http.Request) bool {
	IsAutheticated, ok := r.Context().Value(IsAuthenticatedContextkey).(bool)
	if !ok {
		return false
	}
	return IsAutheticated
}

//
//
//

func (app *Application) ServerError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.Errorlog.Output(2, trace)
	data := &Template{}
	app.Render(w, 500, "servererror.html", data)

}

//
//

func (app *Application) ClientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *Application) RateLimitExcedded(w http.ResponseWriter, r *http.Request) {
	app.ClientError(w, http.StatusTooManyRequests)
}
func (app *Application) NotFound(w http.ResponseWriter) {
	data := &Template{}
	app.Render(w, 404, "notfound.html", data)
}

func (app *Application) NewFormDecoder(r *http.Request, dst any) error {

	err := r.ParseForm()
	if err != nil {
		app.Errorlog.Print(err)
		return err
	}

	err = app.Decoder.Decode(dst, r.PostForm)
	if err != nil {
		var ide *form.InvalidDecoderError
		if errors.As(err, &ide) {
			panic(err)
		}
		return err
	}
	return nil
}

func (app *Application) NewTemplateData(r *http.Request) *Template {

	return &Template{
		Flash:           app.sessionmanager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.IsAuthenticated(r),
		IsActivated:     app.IsActivated(r),
		ShowNav:         true,
	}
}

func (app *Application) NewEmailtempdata(r *http.Request) *EmailTemplates {
	return &EmailTemplates{
		IsAuthenticated: app.IsAuthenticated(r),
		IsActivated:     app.IsActivated(r),
	}
}

func (app *Application) Render(w http.ResponseWriter, status int, name string, data any) {

	ts, ok := app.TempCache[name]
	if !ok {
		app.Errorlog.Println("template not found:", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.Errorlog.Println("template execute error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func getKeys(m map[string]*template.Template) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (app *Application) UploadFileAsync(path, filename string) {
	f, err := os.Open(path)
	if err != nil {
		app.Errorlog.Println(err)
		return
	}
	defer f.Close()
	defer os.Remove(path)

	if err := app.UploadToSupabase(f, filename); err != nil {
		app.Errorlog.Println(err)
	}
}

func (app *Application) UploadToSupabase(file multipart.File, filename string) error {
	url := fmt.Sprintf(
		"%s/storage/v1/object/%s/%s",
		app.Config.supabase.Supabase_Project_url,
		app.Config.supabase.Subase_Storage,
		filename,
	)

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+app.Config.supabase.Supabase_Service_key)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("upload failed: %s", body)
	}

	return nil
}

func (app *Application) pasteout(w http.ResponseWriter, r *http.Request) {
	app.NotFound(w)
}
