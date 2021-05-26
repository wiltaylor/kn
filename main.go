package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	//"github.com/marcusolsson/tui-go"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
  LinkIconUrl = ""
  LinkIconNote = ""
  LinkIconAttachment = ""
)

type LinkType int

const (
  LinkUrl LinkType =  iota
  LinkAttachment
  LinkNote
)

type NoteType int

const (
	Zettle NoteType = iota
	Map
	Literature
	Fleeting
	UnknownType
)

type viewExitResult int

const (
  ViewExitOk viewExitResult = iota
  ViewExitFind
  ViewExitReopen
)

type NoteState int

const (
	New NoteState = iota
	EverGreen
	Done
	UnknownState
)

type LinkRecord struct {
  Title string
  Type LinkType
  Path string
}

type NoteRecord struct {
  Filename string
	Id       string
  Title    string
  Date     string
  Type     NoteType
	State    NoteState
  Links    []LinkRecord
}

type NoteData struct {
  Header NoteRecord
  Text string
}

//TODO: Need to replace this with a yaml parser to make it nice and neat
func GetNoteMeta() []NoteRecord {
	result := make([]NoteRecord, 0)

	files, err := ioutil.ReadDir(getZKPath())

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matched, err := regexp.MatchString(`\.md$`, file.Name())
		if err != nil {
			panic(err)
		}

		if !matched {
			continue
		}

		filePath := filepath.Join(getZKPath(), file.Name())

		fileHandle, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}

		s := bufio.NewScanner(fileHandle)
		s.Scan() // Get rid of first line
		if s.Text() != "---" {
			panic(fmt.Sprintf("Expected first line of %s to be ---", file.Name()))
		}

		curRec := NoteRecord{Title: "", Date: "", Type: UnknownType, State: UnknownState, Filename: filePath, Id: strings.ReplaceAll(file.Name(), ".md", ""), Links: make([]LinkRecord, 0)}

		for s.Scan() {

			txt := s.Text()

			if txt == "---" {
				break
			}

			if strings.HasPrefix(txt, "Title:") {
				curRec.Title = strings.Trim(txt[6:], " ")
				continue
			}

			if strings.HasPrefix(txt, "Date:") {
				curRec.Date = strings.Trim(txt[5:], " ")
				continue
			}

			if strings.HasPrefix(txt, "Type:") {
				noteType := strings.Trim(txt[5:], " ")

				if noteType == "zettle" {
					curRec.Type = Zettle
					continue
				}

				if noteType == "literature" {
					curRec.Type = Literature
					continue
				}

				if noteType == "fleeting" {
					curRec.Type = Fleeting
					continue
				}

				if noteType == "map" {
					curRec.Type = Map
					continue
				}

				curRec.Type = UnknownType
				continue
			}

			if strings.HasPrefix(txt, "Status:") {
				status := strings.Trim(txt[7:], " ")

				if status == "new" {
					curRec.State = New
					continue
				}

				if status == "done" {
					curRec.State = Done
					continue
				}

				if status == "green" {
					curRec.State = EverGreen
					continue
				}

				curRec.State = UnknownState
				continue
			}

      if strings.HasPrefix(txt, "Links:") {
        curLink := LinkRecord{ Title: "", Type: LinkUrl, Path: ""}
        for s.Scan() {
          txt = strings.Trim(s.Text(), " ")
          if len(txt) == 0 {
            break
          }
          if txt[0] == '-' {
            if curLink.Title != "" {
              curRec.Links = append(curRec.Links, curLink)
            }
            txt = txt[2:]
            curLink = LinkRecord{Title: "", Type: LinkUrl, Path: ""}
          }

          if strings.HasPrefix(txt, "Title:") {
            curLink.Title = strings.Trim(txt[6:], " ")
            continue
          }

          if strings.HasPrefix(txt, "Path:") {
            curLink.Path = strings.Trim(txt[5:], " ")
            continue
          }

          if strings.HasPrefix(txt, "Type:") {
            status := strings.Trim(txt[5:], " ")

            if status == "url" {
              curLink.Type = LinkUrl
              continue
            }

            if status == "attachment" {
              curLink.Type = LinkAttachment
              continue
            }

            if status == "note" {
              curLink.Type = LinkNote
              continue
            }

            continue
          }

          break

        }

        if curLink.Title != "" {
          curRec.Links = append(curRec.Links, curLink)
        }
      }
		}

		result = append(result, curRec)
	}

	return result
}

func usage() {
	fmt.Println("kn command [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("new {l,z,f,m}- Create a new note.")
	fmt.Println("   l - literature note")
	fmt.Println("   z - zettle note")
	fmt.Println("   f - fleeting note")
	fmt.Println("   m - map of content note")

}

func getZKPath() string {
	notePath := os.Getenv("ZKDIR")
	if notePath == "" {
		notePath = "./.zk"
	}

	return notePath
}

func newNote(note_type string) {

	notePath := getZKPath()

	switch note_type {
	case "l", "literature":
		note_type = "literature"
	case "z", "zettle":
		note_type = "zettle"
	case "f", "fleeting":
		note_type = "fleeting"
	case "m", "map":
		note_type = "map"
	default:
		panic("Unexpected note type " + note_type + "!")
	}

	curTime := time.Now().UTC()
	atomicId := fmt.Sprintf("%v", curTime.Unix())

	filePath := fmt.Sprintf("%s/%s.md", notePath, atomicId)

	file, err := os.Create(filePath)

	if err != nil {
		panic("Can't create note!")
	}

	writer := bufio.NewWriter(file)
	writer.WriteString("---\n")
	writer.WriteString("Title: New Note\n")
	writer.WriteString("Date: " + curTime.Format(time.RFC822) + "\n")
	writer.WriteString("Type: " + note_type + "\n")
	writer.WriteString("Status: new\n")
	writer.WriteString("---\n")

	writer.Flush()
	file.Close()

  editFile(filePath)

	fmt.Println(atomicId)
}

func searchInNotes(text string, notes *[]NoteRecord, typ NoteType) []NoteRecord {
	result := make([]NoteRecord, 0)

	for _, n := range *notes {
		if n.Type == typ {
			match, err := regexp.MatchString(text, n.Title)

			if err != nil {
				panic(err)
			}

			if match {
        var r NoteRecord = n
				result = append(result, r)
			}
		}
	}

	return result
}

func openNoteData(note NoteRecord) NoteData {
  data, err := ioutil.ReadFile(note.Filename)

  if err != nil {
    panic(err)
  }

  text := string(data)

  index := 0
  remain := 2;
  currentCount := 0

  for i, c := range text {
    if c == '-' {
      currentCount += 1
    }else{
      currentCount = 0
    }

    if currentCount == 3 {
      currentCount = 0
      remain -= 1
    }

    if remain == 0 {
      index = i + 2 // Also eat new line 
      break
    }
  }

  text = text[index:]

  return NoteData{Header: note, Text: text}

}

func editFile(path string) {
  editor := os.Getenv("EDITOR")

	cmd := exec.Command(editor, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

  err := cmd.Run()

	if err != nil {
		panic(err)
	}

}

func viewUI(doc NoteData) viewExitResult{
  app := tview.NewApplication()
  textView := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetWordWrap(true).SetChangedFunc(func() {
    app.Draw()
  })

  textView.SetText(doc.Text)
  textView.SetScrollable(true)
  textView.SetTitle(doc.Header.Title)
  textView.SetBorder(true)
  textView.SetBorderPadding(0,0,1,1)
  app.SetFocus(textView)
  exitCode := ViewExitOk

  toolBar := tview.NewTextView()
  toolBar.SetText("ESC - Quit | F - Find | E - Edit | Arrows - Move")
  toolBar.SetBackgroundColor(tcell.ColorWhite)
  toolBar.SetTextColor(tcell.ColorBlack)

  links := tview.NewList()
  links.SetTitle("Links")
  links.SetBorder(true)
  links.ShowSecondaryText(false)
  links.SetDoneFunc(func() {
    app.Stop()
  })

  for _, i := range doc.Header.Links {
    linkIcon := LinkIconUrl

    if i.Type == LinkNote {
      linkIcon = LinkIconNote
    }

    if i.Type == LinkAttachment {
      linkIcon = LinkIconAttachment
    }

    links.AddItem(linkIcon + " " + i.Title , "", '\n', nil)
  }

  layout := tview.NewGrid()
  layout.SetRows(1, 5, 0)
  layout.AddItem(toolBar, 0, 0, 1, 1, 1, 1, false)
  layout.AddItem(textView, 1, 0, 10, 1, 10, 40, true)
  layout.AddItem(links, 11, 0, 1, 1, 1, 40, false)

	app.SetInputCapture(func(key *tcell.EventKey) *tcell.EventKey {

		if key.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}

    if key.Key() == tcell.KeyEnter {
      if links.HasFocus() {
        linkIndex := links.GetCurrentItem()
        lnk := doc.Header.Links[linkIndex]

        if lnk.Type == LinkUrl {
          cmd := exec.Command("xdg-open", lnk.Path)
          cmd.Run()
        }

        if lnk.Type == LinkAttachment {

        }

        return nil
      }
    }

    if key.Key() == tcell.KeyTab {
      if textView.HasFocus() {
        app.SetFocus(links)
        return nil
      }

      if links.HasFocus() {
        app.SetFocus(textView)
        return nil
      }
    }

    if key.Rune() == 'e' {
      editFile(doc.Header.Filename)
      exitCode = ViewExitReopen
      app.Stop()
      return nil
    }

    if key.Rune() == 'f' {
      exitCode = ViewExitFind
      app.Stop()
      return nil
    }

    return key
  })

  if err := app.SetRoot(layout, true).Run(); err != nil {
    panic(err)
  }

  return exitCode
}

func searchUI(text string) *NoteRecord {

	allNotes := GetNoteMeta()
	selectedNoteType := Zettle
  var selectedNote *NoteRecord
  selectedNote = nil

	app := tview.NewApplication()

	searchResult := tview.NewList()
	searchResult.ShowSecondaryText(false)
	searchField := tview.NewInputField().SetLabel("Search")
	searchField.SetText(text)

	searchNoteType := tview.NewDropDown()

	searchResult.Clear()

	{
		notes := searchInNotes(text, &allNotes, selectedNoteType)
		for _, n := range notes {
			searchResult.AddItem(n.Title, "", '\n', func() {
				selectedNote = &n
				app.Stop()
			})
		}
	}

	searchNoteType.SetOptions([]string{"Zettle", "Litrature", "Fleeting", "Map"}, func(text string, index int) {

	})

	searchNoteType.SetLabel("Note Type: ").SetCurrentOption(0)

	app.SetInputCapture(func(key *tcell.EventKey) *tcell.EventKey {

		if key.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}

    if key.Key() == tcell.KeyEnter {
      idx := searchResult.GetCurrentItem()

      app.Stop()
      return nil
    }

		if key.Key() == tcell.KeyTab {
			if searchField.HasFocus() {
				app.SetFocus(searchResult)
				return nil
			}

			if searchResult.HasFocus() {
				app.SetFocus(searchNoteType)
				return nil
			}

			if searchNoteType.HasFocus() {
				app.SetFocus(searchField)
				return nil
			}
		}

		return key
	})

	searchField.SetChangedFunc(func(text string) {
		notes := searchInNotes(text, &allNotes, selectedNoteType)

		searchResult.Clear()

		for _, n := range notes {
			searchResult.AddItem(n.Title, "", '\n', nil)
		}

	})

	grid := tview.NewGrid()
	grid.SetRows(1, 1, 0)
	grid.SetBorders(true)
	grid.SetMinSize(0, 0)
	grid.AddItem(searchField, 0, 0, 1, 1, 1, 40, true)
	grid.AddItem(searchNoteType, 1, 0, 1, 1, 1, 40, false)
	grid.AddItem(searchResult, 2, 0, 10, 1, 10, 40, false)

	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}

	return selectedNote
}

func main() {
	flag.Parse()
	cmd := flag.Arg(0)

	if flag.NArg() < 2 {
		usage()
		return
	}

	switch cmd {
	case "search":
    result :=	searchUI(flag.Arg(1))

    if result == nil {
      fmt.Println("Nothing selected")
      return
    }

    vresult := ViewExitReopen

    for {
      vresult = viewUI(openNoteData(*result))

      if vresult == ViewExitOk {
        break
      }

      if vresult == ViewExitFind {
        result = searchUI("")
        if result == nil {
          break
        }
      }
    }

	case "new":
		newNote(flag.Arg(1))
	default:
		usage()
	}
}
