package model

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Post struct {
	Id        int32
	Userid    int32
	UserName  string
	TiTle     string
	Imageurl  string
	PdfFile   string
	Content   string
	Category  string
	Createdat time.Time
	Version   int
	Rating    float64
	Comments  []*Comment
}
type Comment struct {
	CUserid   int16
	CUserName string
	CPostid   int16
	Comment   string
}

type PostModel struct {
	DB *sql.DB
}

func (m *PostModel) Insert(title, content, category string, Imageurl sql.NullString, pdfFile string, Userid int) (int, error) {
	var id int
	query := `INSERT INTO posts (userid, title, content, category , imgaddr, pdffile) 
	VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	err := m.DB.QueryRow(query, Userid, title, content, category, Imageurl, pdfFile).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil

}

func (m *PostModel) Get(id int) (*Post, error) {

	var Imageaddr sql.NullString

	query := `SELECT posts.id, userid, users.name, title, content, category, imgaddr, pdffile FROM posts JOIN users ON posts.userid = users.id 
	WHERE posts.id = $1 `
	s := &Post{}
	row := m.DB.QueryRow(query, id)
	err := row.Scan(&s.Id, &s.Userid, &s.UserName, &s.TiTle, &s.Content, &s.Category, &Imageaddr, &s.PdfFile)
	if Imageaddr.Valid {
		s.Imageurl = Imageaddr.String
	} else {
		s.Imageurl = ""
	}
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrecordNotFound
		default:
			return nil, err
		}
	}
	comm, err := m.GetComments(id)
	if err != nil {
		return nil, err
	}

	s.Comments = comm

	return s, nil

}

func (m *PostModel) Latest() ([]*Post, error) {

	//query := `SELECT userid, users.name, posts.id, title, content, imgaddr FROM posts JOIN users ON  users.id = posts.userid ORDER BY posts.id DESC LIMIT 10`
	query := `SELECT posts.userid, users.name, posts.id, Round(AVG(ratings.rating),1), title, content, imgaddr 
	FROM posts Left JOIN
ratings ON ratings.postid = posts.id 
Join users ON  users.id = posts.userid
GROUP BY 
posts.userid, users.name, posts.id, title , content, imgaddr
ORDER BY posts.id DESC LIMIT 10`
	var imgaddres sql.NullString
	var rating sql.NullFloat64
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []*Post{}

	for rows.Next() {
		s := &Post{}
		err := rows.Scan(&s.Userid, &s.UserName, &s.Id, &rating, &s.TiTle, &s.Content, &imgaddres)
		if rating.Valid {
			s.Rating = rating.Float64
		} else {
			s.Rating = float64(0)
		}
		if imgaddres.Valid {
			s.Imageurl = imgaddres.String
		} else {
			s.Imageurl = ""
		}
		if err != nil {
			fmt.Print("rows.scan errror ")
			return nil, err

		}

		results = append(results, s)
	}
	return results, nil
}

func (m *PostModel) UpdateDownloadCounts(userid int) error {
	stmt := `INSERT INTO downloads (userid, downloadattempts)
	VALUES ($1, 1)
	ON CONFLICT (userid)
	DO UPDATE SET downloadattempts = downloads.downloadattempts +1`

	_, err := m.DB.Exec(stmt, userid)
	if err != nil {
		return err
	}
	return nil
}

func (m *PostModel) ReturnDownloadCounts(userid int) (int, error) {

	var attempt int

	query := `SELECT downloadattempts FROM downloads WHERE userid = $1`

	err := m.DB.QueryRow(query, userid).Scan(&attempt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, ErrecordNotFound
		default:
			return 0, err
		}
	}

	return attempt, nil

}

func (m *PostModel) GetComments(id int) ([]*Comment, error) {

	qury := `SELECT users.name , users.id, comments.content FROM comments JOIN users ON 
	users.id = comments.userid 
	INNER JOIN posts ON comments.postid = posts.id 
	WHERE posts.id = $1`

	rows, err := m.DB.Query(qury, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []*Comment{}

	for rows.Next() {

		comment := &Comment{}
		err := rows.Scan(&comment.CUserName, &comment.CUserid, &comment.Comment)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)

	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *PostModel) INsertComment(content string, userid, postid int) (int, error) {

	query := `INSERT INTO comments (content, userid, postid) VALUES ($1, $2, $3) RETURNING id`
	var id int
	err := m.DB.QueryRow(query, content, userid, postid).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil

}

func (m *PostModel) InsertRating(userid, postid int, rating any) error {

	query := `INSERT INTO ratings(userid, postid, rating) VALUES ($1, $2, $3)`

	_, err := m.DB.Exec(query, userid, postid, rating)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "idx_unique"`:
			return ErrDuplicateemail
		default:
			return err
		}
	}
	return nil

}

func (m *PostModel) CheckRated(userid, postid int) (bool, error) {
	var Exists bool
	query := `SELECT EXISTS(SELECT true FROM ratings WHERE userid = $1 AND postid = $2)`

	err := m.DB.QueryRow(query, userid, postid).Scan(&Exists)
	if err != nil {
		return false, err
	}

	return Exists, nil

}

func (m *PostModel) GetPdfUrl(postid int) (string, error) {

	var pdfurl string

	stmt := `SELECT pdffile FROM posts WHERE posts.id = $1`

	err := m.DB.QueryRow(stmt, postid).Scan(&pdfurl)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrecordNotFound
		default:
			return "", err
		}

	}

	return pdfurl, nil

}

func (m *PostModel) GetPostOfAUser(id int) ([]*Post, error) {

	stmt := `SELECT posts.id, posts.title, posts.created_at FROM posts
WHERE posts.userid = $1
 ORDER BY id DESC LIMIT 10`

	rows, err := m.DB.Query(stmt, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrecordNotFound
		default:
			return nil, err
		}

	}

	defer rows.Close()

	posts := []*Post{}

	for rows.Next() {
		post := &Post{}

		err = rows.Scan(&post.Id, &post.TiTle, &post.Createdat)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil

}

func (m *PostModel) SearchPosts(search string) ([]*Post, error) {

	query := `SELECT id, title, created_at FROM posts WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1))
	LIMIT 10`

	rows, err := m.DB.Query(query, search)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrecordNotFound
		default:
			return nil, err
		}

	}

	defer rows.Close()

	posts := []*Post{}

	for rows.Next() {
		singlePost := &Post{}

		err := rows.Scan(&singlePost.Id, &singlePost.TiTle, &singlePost.Createdat)
		if err != nil {
			return nil, err
		}

		posts = append(posts, singlePost)
	}

	return posts, nil

}
