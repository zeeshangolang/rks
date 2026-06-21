package main

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/Backblaze/blazer/b2"
	"github.com/joho/godotenv"
)

func Bbintialize() *b2.Bucket {

	err := godotenv.Load()
	if err != nil {
		fmt.Print("error loading env ")
	}

	id := os.Getenv("BBKEYID")
	key := os.Getenv("BBAPPKEY")

	ctx := context.Background()

	//authorize account

	b2, err := b2.NewClient(ctx, id, key)
	if err != nil {
		fmt.Print(err)
		return nil
	}

	buckets, err := b2.ListBuckets(ctx)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	fmt.Print("this ->", buckets)

	return buckets[0]
}

//func (b)

func (app *Application) seee() {

	ctx := context.Background()

	ggg, err := app.Bucket.Attrs(ctx)

	if err != nil {
		print(err)
	}

	fmt.Print(*ggg)
}

func (app *Application) BaseUrl() string {

	baseurll := app.Bucket.BaseURL()

	return baseurll

}

func (app *Application) Name() string {

	name := app.Bucket.Name()

	return name
}

func (app *Application) CopyFile(ctx context.Context, bucket *b2.Bucket, f multipart.File, dest string) error {

	obj := bucket.Object(dest)
	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, f); err != nil {
		w.Close()
		return err
	}

	return w.Close()

}
