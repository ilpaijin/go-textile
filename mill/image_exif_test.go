package mill

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/textileio/textile-go/mill/testdata"
)

func TestImageExif_Mill(t *testing.T) {
	m := &ImageExif{}

	for _, i := range testdata.Images {
		file, err := os.Open(i.Path)
		if err != nil {
			t.Fatal(err)
		}

		input, err := ioutil.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()

		res, err := m.Mill(input, "test")
		if err != nil {
			t.Fatal(err)
		}

		var exif *ImageExifSchema
		if err := json.Unmarshal(res.File, &exif); err != nil {
			t.Fatal(err)
		}

		if exif.Width != i.Width {
			t.Errorf("wrong width")
		}
		if exif.Height != i.Height {
			t.Errorf("wrong height")
		}
		if exif.Format != i.Format {
			t.Errorf("wrong format")
		}
		if exif.Created.Unix() != i.Created {
			t.Error("wrong created")
		}
		if (i.HasExif && exif.Latitude == 0) || (!i.HasExif && exif.Latitude != 0) {
			t.Error("wrong latitude")
		}
		if (i.HasExif && exif.Longitude == 0) || (!i.HasExif && exif.Longitude != 0) {
			t.Error("wrong longitude")
		}
	}
}
