
package main 

import (
  "fmt"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"os/exec"
	"io"
	"bufio"
	"strings"
	"os"
	"os/user"
)

// helper
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Radio Station
type RadioStation struct {
	name string
	stream_url  string
}

// Dj
type Dj struct {
	player RadioPlayer
	stations []RadioStation
	current_station int
}

func (dj *Dj) Play(station int) {
	if 0 <= station && station < len(dj.stations)  && dj.current_station != station {
		if (dj.current_station >= 0) {
			dj.player.Close()
		}

		dj.current_station = station
		dj.player.Play(dj.stations[dj.current_station].stream_url)
	}
}

func (dj *Dj) Stop() {
	if dj.current_station >= 0 {
		dj.player.Close()
		dj.current_station = -1
	}
}

func (dj *Dj) Mute() {
	if dj.current_station >= 0 {
		dj.player.Mute()
	}
}

func (dj *Dj) Turnup() {
	if dj.current_station >= 0 {
		dj.player.IncVolume()
	}
}

func (dj *Dj) Turndown() {
	if dj.current_station >= 0 {
		dj.player.DecVolume()
	}
}

// Radio player interface
type RadioPlayer interface {
	Play(stream_url string)
	Mute()
	Pause()
	IncVolume()
	DecVolume()
	Close()
}

// MPlayer
type MPlayer struct {
	player_name string
	is_playing  bool
	stream_url  string
	command     *exec.Cmd
	in          io.WriteCloser
	out         io.ReadCloser
	pipe_chan   chan io.ReadCloser
}

func (player *MPlayer) Play(stream_url string) {
	if !player.is_playing {
		var err error
		is_playlist := strings.HasSuffix(stream_url, ".m3u") || strings.HasSuffix(stream_url, ".pls")
		if is_playlist {
			player.command = exec.Command(player.player_name, "-quiet", "-playlist", stream_url)
		} else {
			player.command = exec.Command(player.player_name, "-quiet", stream_url)
		}
		player.in, err = player.command.StdinPipe()
		check(err)
		player.out, err = player.command.StdoutPipe()
		check(err)

		err = player.command.Start()
		check(err)
		
		player.is_playing = true
		player.stream_url = stream_url
		go func() {
			player.pipe_chan<- player.out	
		}()
	} 
}

func (player *MPlayer) Close() {
	if player.is_playing {
		player.is_playing = false

		player.in.Write([]byte("q"))	
		player.in.Close()
		player.out.Close()
		player.command = nil

		player.stream_url = ""
	}
}

func (player *MPlayer) Mute() {
	if player.is_playing {
		player.in.Write([]byte("m"))	
	}
}

func (player *MPlayer) Pause() {
	if player.is_playing {
		player.in.Write([]byte("p"))	
	}
}

func (player *MPlayer) IncVolume() {
	if player.is_playing {
		player.in.Write([]byte("*"))	
	}
}

func (player *MPlayer) DecVolume() {
	if player.is_playing {
		player.in.Write([]byte("/"))	
	}
}

func draw_horizontal_line(x1, x2, y int, fg termbox.Attribute, bg termbox.Attribute, ch rune) {
	for x := x1; x <= x2; x++ {
		termbox.SetCell(x, y, ch, fg, bg)
	}	
}

func draw_borders(w, h int) {
	draw_horizontal_line(1, w, 0, termbox.ColorDefault, termbox.ColorDefault, '-')
	draw_horizontal_line(1, w, 2, termbox.ColorDefault, termbox.ColorDefault, '-')
	draw_horizontal_line(1, w, h-2, termbox.ColorDefault, termbox.ColorDefault, '-')
}

func draw_header(w, h int) {
	var banner = "GoRadio"
	var link = "github.com/jcheng8/goradio"

	fmt.Sprintf("%v %v", w-len(link), h)

	print_tb(1, 1,             termbox.ColorDefault, termbox.ColorDefault, banner)
	print_tb(w-1-len(link), 1, termbox.ColorDefault, termbox.ColorDefault, link)
}

func draw_footer(msg string) {
	w, h := termbox.Size()
	y := h - 1
	print_tb(1, y, termbox.ColorDefault, termbox.ColorDefault, msg)
	x := 1 + len(msg)
	fill(x, y, w, termbox.ColorDefault, termbox.ColorDefault, ' ')
}

func draw_stations(stations []RadioStation, cursor_on_station int) {
	w, h := termbox.Size()
	top := 3
	bottom := h - 2
	allowed_h := bottom - top 
	if allowed_h <= 0 {
		return
	}
	
	total_stations := len(stations)
	start := 0
	end   := total_stations - 1

	if (total_stations > allowed_h) {
		if (cursor_on_station >= allowed_h) {
			start = cursor_on_station - allowed_h + 1
			end   = cursor_on_station
		} else {
			start = 0
			end   = allowed_h - 1
		}
	}

  y := top
	for x := start; x <= end; x++ {
		fg, bg := termbox.ColorDefault, termbox.ColorDefault
		if x == cursor_on_station {
			fg, bg = termbox.ColorRed, termbox.ColorBlack
		}
		station := stations[x]
		print_tb(1, y, fg, bg, station.name)

		fill(len(station.name) + 1, y, w, fg, bg, ' ')
		y++
	}
}

func draw_all(stations []RadioStation, cursor_on_station int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	w, h := termbox.Size()
	draw_header(w, h)
	draw_footer("Ready (Esc: Quit app | Enter: Play | q: Stop | m: Mute | +: Louder | -: Quieter | k/↑ : Up | j/↓: Down)")
	draw_stations(stations, cursor_on_station)
	draw_borders(w, h)

	termbox.Flush()
}

func print_tb(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range []rune(msg) {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func fill(x1, y, x2 int, fg, bg termbox.Attribute, ch rune) {
	for i := x1; i < x2; i++ {
		termbox.SetCell(i, y, ch, fg, bg)
	}
}

func load_stations() []RadioStation {
	var stations []RadioStation

	usr, _ := user.Current()
	dir := usr.HomeDir
	default_file := dir + "/.goradio/stations"
	if Exists(default_file) {
		f, err := os.Open(default_file)
		check(err)
		defer f.Close()

		scanner:= bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.Trim(scanner.Text(), "\n\r")
			pair := strings.Split(line, ",")
			if len(pair) == 2 {
				stations = append(stations, 
					                RadioStation{strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])})
			}
		}
		check(scanner.Err())
	} else {
		stations = append(stations, RadioStation{"WBEZ 91.5", "http://stream.wbez.org/wbez128.mp3"})	
		stations = append(stations, RadioStation{"WGN", "http://provisioning.streamtheworld.com/pls/WGNPLUSAM.pls"})
	}
	return stations
}

func Exists(name string) bool {
    if _, err := os.Stat(name); err != nil {
	    if os.IsNotExist(err) {return false}
    }
    return true
}

func main() {
	var stations = load_stations()

	var num_of_stations = len(stations)

	if num_of_stations == 0 {
		panic("no stations")
	}

	var status_chan = make(chan string)
	var	pipe_chan = make(chan io.ReadCloser)
	var mplayer = MPlayer{player_name: "mplayer", is_playing: false, pipe_chan: pipe_chan}

	david := Dj{player: &mplayer, stations: stations, current_station: -1}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.HideCursor()

	event_queue := make(chan termbox.Event)

	go func() {
		for {
			event_queue <- termbox.PollEvent()
		}
	}()

	go func() {
		for {
			out_pipe := <-pipe_chan
	    reader := bufio.NewReader(out_pipe)
	    for {
				data, err := reader.ReadString('\n')
				if err != nil {
					status_chan<- "Playing stopped"
					break
				} else {
					status_chan<- data
				}
			}
		}
	}()

	cursor_on_station := 0

	draw_all(stations, cursor_on_station)

	loop:
		for {
			select {
				case process_output := <-status_chan:
					draw_footer(process_output)
					termbox.Flush()
				case ev := <-event_queue:
					switch ev.Type {
						case termbox.EventKey:
							switch ev.Key {
								case termbox.KeyEsc:
									break loop
								case termbox.KeyEnter:
									david.Play(cursor_on_station)
								default:
									if ev.Ch == 'q' {
										david.Stop()
									}
									if ev.Ch == '+' {
										david.Turnup()
									}
									if ev.Ch == '-' {
										david.Turndown()
									}
									if ev.Ch == 'm' {
										david.Mute()
									}
									if ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown {
										if cursor_on_station < num_of_stations - 1 {
											cursor_on_station += 1
											draw_stations(stations, cursor_on_station)
											termbox.Flush()
										}
									}
									if ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp {
										if cursor_on_station > 0 {
											cursor_on_station -= 1
											draw_stations(stations, cursor_on_station)
											termbox.Flush()
										}
									}
								}
						case termbox.EventResize:
							draw_all(stations, cursor_on_station)
						case termbox.EventInterrupt:
							break loop
						case termbox.EventError:
							panic(ev.Err)
					}
			}
		}

	david.Stop()
	close(event_queue)
	close(status_chan)
	close(pipe_chan)
}