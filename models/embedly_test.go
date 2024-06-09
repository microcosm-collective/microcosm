package models

import "testing"

func TestEmbedly(t *testing.T) {
	src := `<a href="https://www.youtube.com/watch?v=hnRdIMklPow">A youtube video</a>`
	expected := `<a href="https://www.youtube.com/watch?v=hnRdIMklPow">A youtube video</a><br />
<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/hnRdIMklPow" frameborder="0" allowfullscreen></iframe>`

	out := string(Embedly([]byte(src)))

	if out != expected {
		t.Errorf("Expected: %s\nGot     : %s", expected, out)
	}

	src = `<a href="https://www.example.org/">Example</a>`
	expected = src

	out = string(Embedly([]byte(src)))

	if out != expected {
		t.Errorf("Expected: %s\nGot     : %s", expected, out)
	}

	src = `<a href="https://www.youtube.com/watch?v=hnRdIMklPo1">A youtube video</a><a href="https://www.youtube.com/watch?v=hnRdIMklPo2">A youtube video</a>`
	expected = `<a href="https://www.youtube.com/watch?v=hnRdIMklPo1">A youtube video</a><br />
<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/hnRdIMklPo1" frameborder="0" allowfullscreen></iframe><a href="https://www.youtube.com/watch?v=hnRdIMklPo2">A youtube video</a><br />
<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/hnRdIMklPo2" frameborder="0" allowfullscreen></iframe>`

	out = string(Embedly([]byte(src)))

	if out != expected {
		t.Errorf("Expected: %s\nGot     : %s", expected, out)
	}

}
