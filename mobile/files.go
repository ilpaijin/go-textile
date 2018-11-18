package mobile

import (
	"errors"
	"io"
	"io/ioutil"
	"os"

	m "github.com/textileio/textile-go/mill"
	"github.com/textileio/textile-go/repo"
	"github.com/textileio/textile-go/schema"

	"github.com/textileio/textile-go/core"
)

// FileData is a wrapper around a file data url
type FileData struct {
	Url string `json:"url"`
}

// AddFile processes a file by path for a thread, but does NOT share it
func (m *Mobile) AddFile(path string, threadId string) (string, error) {
	thrd := m.node.Thread(threadId)
	if thrd == nil {
		return "", core.ErrThreadNotFound
	}

	if thrd.Schema == nil {
		return "", core.ErrThreadSchemaRequired
	}

	var result interface{}

	mill, err := getMill(thrd.Schema.Mill, thrd.Schema.Opts)
	if err != nil {
		return "", err
	}
	if mill != nil {
		conf, err := m.getFileConfig(mill, path, "")
		if err != nil {
			return "", err
		}

		added, err := m.node.AddFile(mill, *conf)
		if err != nil {
			return "", err
		}
		result = &added

	} else if len(thrd.Schema.Links) > 0 {
		dir := make(map[string]*repo.File)

		// determine order
		steps, err := schema.Steps(thrd.Schema.Links)
		if err != nil {
			return "", err
		}

		// send each link
		for _, step := range steps {
			mill, err := getMill(step.Link.Mill, step.Link.Opts)
			if err != nil {
				return "", err
			}
			var conf *core.AddFileConfig

			if step.Link.Use == schema.FileTag {
				conf, err = m.getFileConfig(mill, path, "")
				if err != nil {
					return "", err
				}

			} else {
				if dir[step.Link.Use] == nil {
					return "", errors.New(step.Link.Use + " not found")
				}
				conf, err = m.getFileConfig(mill, path, dir[step.Link.Use].Hash)
				if err != nil {
					return "", err
				}
			}
			added, err := m.node.AddFile(mill, *conf)
			if err != nil {
				return "", err
			}
			dir[step.Name] = added
		}
		result = &dir

	} else {
		return "", schema.ErrEmptySchema
	}

	return toJSON(result)
}

func (m *Mobile) getFileConfig(mill m.Mill, path string, use string) (*core.AddFileConfig, error) {
	var reader io.ReadSeeker
	conf := &core.AddFileConfig{}

	if use == "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		reader = f
		conf.Name = f.Name()
	} else {
		var file *repo.File
		var err error
		reader, file, err = m.node.FilePlaintext(use)
		if err != nil {
			return nil, err
		}
		conf.Name = file.Name
		conf.Use = file.Checksum
	}

	media, err := m.node.GetMedia(reader, mill)
	if err != nil {
		return nil, err
	}
	conf.Media = media
	reader.Seek(0, 0)

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	conf.Input = data

	return conf, nil
}

func getMill(id string, opts map[string]string) (m.Mill, error) {
	switch id {
	case "/blob":
		return &m.Blob{}, nil
	case "/image/resize":
		width := opts["width"]
		if width == "" {
			return nil, errors.New("missing width")
		}
		quality := opts["quality"]
		if quality == "" {
			quality = "75"
		}
		return &m.ImageResize{
			Opts: m.ImageResizeOpts{
				Width:   width,
				Quality: quality,
			},
		}, nil
	case "/image/exif":
		return &m.ImageExif{}, nil
	default:
		return nil, nil
	}
}

//// AddPhotoToThread adds an existing photo to a new thread
//func (m *Mobile) AddPhotoToThread(dataId string, key string, threadId string, caption string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	thrd := m.node.Thread(threadId)
//	if thrd == nil {
//		return "", core.ErrThreadNotFound
//	}
//	keyb, err := base58.Decode(key)
//	if err != nil {
//		return "", err
//	}
//	hash, err := thrd.AddFile(dataId, caption, keyb)
//	if err != nil {
//		return "", err
//	}
//	return hash.B58String(), nil
//}
//
//// SharePhotoToThread adds an existing photo to a new thread
//func (m *Mobile) SharePhotoToThread(dataId string, threadId string, caption string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	block, err := m.node.BlockByDataId(dataId)
//	if err != nil {
//		return "", err
//	}
//	toThread := m.node.Thread(threadId)
//	if toThread == nil {
//		return "", core.ErrThreadNotFound
//	}
//	// owner challenge
//	hash, err := toThread.AddFile(dataId, caption, block.DataKey)
//	if err != nil {
//		return "", err
//	}
//	return hash.B58String(), nil
//}
//
//// Photos returns thread photo blocks with json encoding
//func (m *Mobile) Photos(offset string, limit int, threadId string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	var pre, query string
//	if threadId != "" {
//		thrd := m.node.Thread(threadId)
//		if thrd == nil {
//			return "", core.ErrThreadNotFound
//		}
//		pre = fmt.Sprintf("threadId='%s' and ", threadId)
//	}
//	query = fmt.Sprintf("%stype=%d", pre, repo.FilesBlock)
//
//	// build json
//	photos := &Photos{Items: make([]Photo, 0)}
//	for _, b := range m.node.Blocks(offset, limit, query) {
//		item := Photo{
//			Id:       b.DataId,
//			BlockId:  b.Id,
//			Date:     b.Date,
//			AuthorId: b.AuthorId,
//			Caption:  b.DataCaption,
//			Username: m.node.ContactUsername(b.AuthorId),
//			Metadata: b.DataMetadata,
//		}
//
//		// add comments
//		cquery := fmt.Sprintf("%stype=%d and dataId='%s'", pre, repo.CommentBlock, b.Id)
//		item.Comments = make([]Comment, 0)
//		for _, c := range m.node.Blocks("", -1, cquery) {
//			comment := Comment{
//				Annotation: Annotation{
//					Id:       c.Id,
//					Date:     c.Date,
//					AuthorId: c.AuthorId,
//					Username: m.node.ContactUsername(c.AuthorId),
//				},
//				Body: c.DataCaption,
//			}
//			item.Comments = append(item.Comments, comment)
//		}
//
//		// add likes
//		lquery := fmt.Sprintf("%stype=%d and dataId='%s'", pre, repo.LikeBlock, b.Id)
//		item.Likes = make([]Like, 0)
//		for _, l := range m.node.Blocks("", -1, lquery) {
//			like := Like{
//				Annotation: Annotation{
//					Id:       l.Id,
//					Date:     l.Date,
//					AuthorId: l.AuthorId,
//					Username: m.node.ContactUsername(l.AuthorId),
//				},
//			}
//			item.Likes = append(item.Likes, like)
//		}
//
//		// collect
//		photos.Items = append(photos.Items, item)
//	}
//	return toJSON(photos)
//}
//
//// IgnorePhoto is a semantic helper for mobile, just calls IgnoreBlock
//func (m *Mobile) IgnorePhoto(blockId string) (string, error) {
//	return m.ignoreBlock(blockId)
//}
//
//// AddPhotoComment adds an comment block targeted at the given block
//func (m *Mobile) AddPhotoComment(blockId string, body string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	block, err := m.node.Block(blockId)
//	if err != nil {
//		return "", err
//	}
//	thrd := m.node.Thread(block.ThreadId)
//	if thrd == nil {
//		return "", core.ErrThreadNotFound
//	}
//	hash, err := thrd.AddComment(block.Id, body)
//	if err != nil {
//		return "", err
//	}
//	return hash.B58String(), nil
//}
//
//// IgnorePhotoComment is a semantic helper for mobile, just call IgnoreBlock
//func (m *Mobile) IgnorePhotoComment(blockId string) (string, error) {
//	return m.ignoreBlock(blockId)
//}
//
//// AddPhotoLike adds a like block targeted at the given block
//func (m *Mobile) AddPhotoLike(blockId string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	block, err := m.node.Block(blockId)
//	if err != nil {
//		return "", err
//	}
//	thrd := m.node.Thread(block.ThreadId)
//	if thrd == nil {
//		return "", core.ErrThreadNotFound
//	}
//	hash, err := thrd.AddLike(block.Id)
//	if err != nil {
//		return "", err
//	}
//	return hash.B58String(), nil
//}
//
//// IgnorePhotoLike is a semantic helper for mobile, just call IgnoreBlock
//func (m *Mobile) IgnorePhotoLike(blockId string) (string, error) {
//	return m.ignoreBlock(blockId)
//}
//
//// PhotoData returns a data url of an image under a path
//func (m *Mobile) PhotoData(id string, path string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	block, err := m.node.BlockByDataId(id)
//	if err != nil {
//		return "", err
//	}
//	data, err := m.node.BlockData(fmt.Sprintf("%s/%s", id, path), block)
//	if err != nil {
//		return "", err
//	}
//	format := block.DataMetadata.EncodingFormat
//	prefix := getImageDataURLPrefix(images.Format(format))
//	encoded := libp2pc.ConfigEncodeKey(data)
//	img := &ImageData{Url: prefix + encoded}
//	return toJSON(img)
//}
//
//// PhotoDataForSize returns a data url of an image at or above requested size, or the next best option
//func (m *Mobile) PhotoDataForMinWidth(id string, minWidth int) (string, error) {
//	path := images.ImagePathForSize(images.ImageSizeForMinWidth(minWidth))
//	return m.PhotoData(id, string(path))
//}
//
//// PhotoMetadata returns a meta data object for a photo
//func (m *Mobile) PhotoMetadata(id string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	block, err := m.node.BlockByDataId(id)
//	if err != nil {
//		return "", err
//	}
//	return toJSON(block.DataMetadata)
//}
//
//// PhotoKey calls core PhotoKey
//func (m *Mobile) PhotoKey(id string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	key, err := m.node.PhotoKey(id)
//	if err != nil {
//		return "", err
//	}
//	return base58.FastBase58Encoding(key), nil
//}
//
//// PhotoThreads call core PhotoThreads
//func (m *Mobile) PhotoThreads(id string) (string, error) {
//	if !m.node.Started() {
//		return "", core.ErrStopped
//	}
//	threads := Threads{Items: make([]Thread, 0)}
//	for _, thrd := range m.node.PhotoThreads(id) {
//		peers := thrd.Peers()
//		item := Thread{Id: thrd.Id, Name: thrd.Name, Peers: len(peers)}
//		threads.Items = append(threads.Items, item)
//	}
//	return toJSON(threads)
//}

// ignoreBlock adds an ignore block targeted at the given block and unpins any associated block data
func (m *Mobile) ignoreBlock(blockId string) (string, error) {
	if !m.node.Started() {
		return "", core.ErrStopped
	}
	block, err := m.node.Block(blockId)
	if err != nil {
		return "", err
	}
	thrd := m.node.Thread(block.ThreadId)
	if thrd == nil {
		return "", core.ErrThreadNotFound
	}
	hash, err := thrd.AddIgnore(block.Id)
	if err != nil {
		return "", err
	}
	return hash.B58String(), nil
}

// getImageDataURLPrefix adds the correct data url prefix to a data url
//func getImageDataURLPrefix(format images.Format) string {
//	switch format {
//	case images.PNG:
//		return "data:image/png;base64,"
//	case images.GIF:
//		return "data:image/gif;base64,"
//	default:
//		return "data:image/jpeg;base64,"
//	}
//}

//// FileThreads lists threads which contain a photo (known to the local peer)
//func (t *Textile) FileThreads(id string) []*Thread {
//	blocks := t.datastore.Blocks().List("", -1, "dataId='"+id+"'")
//	if len(blocks) == 0 {
//		return nil
//	}
//	var threads []*Thread
//	for _, block := range blocks {
//		if thrd := t.Thread(block.ThreadId); thrd != nil {
//			threads = append(threads, thrd)
//		}
//	}
//	return threads
//}
