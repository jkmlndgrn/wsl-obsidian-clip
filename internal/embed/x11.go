package embed

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Writer offers embed text on the X11 clipboard and saves the attachment lazily
// when the embed text is first requested by a paste target.
type Writer interface {
	OfferEmbed(filename string, onPaste func() error) error
	Close() error
}

type atoms struct {
	atom      xproto.Atom
	clipboard xproto.Atom
	string    xproto.Atom
	targets   xproto.Atom
	text      xproto.Atom
	utf8      xproto.Atom
	wmClass   xproto.Atom
	wmName    xproto.Atom
	netWmName xproto.Atom
}

type offer struct {
	filename string
	text     string
	onPaste  func() error
	saved    bool
	offeredAt time.Time
}

type requestorInfo struct {
	class string
	name  string
}

const (
	initialProbeWindow     = 500 * time.Millisecond
	unknownRequestorWindow = 2 * time.Second
)

// X11Writer owns the X11 clipboard directly so it can save the file on the
// first actual paste request instead of when the screenshot is detected.
type X11Writer struct {
	conn   *xgb.Conn
	window xproto.Window
	atoms  atoms
	logger *log.Logger

	mu      sync.Mutex
	current *offer
	done    chan struct{}
}

// NewX11Writer creates a clipboard owner for lazy embed delivery.
func NewX11Writer(logger *log.Logger) (*X11Writer, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connect to X11: %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	window, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("allocate X11 window: %w", err)
	}

	if err := xproto.CreateWindowChecked(
		conn,
		screen.RootDepth,
		window,
		screen.Root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,
		0,
		nil,
	).Check(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create X11 window: %w", err)
	}

	writer := &X11Writer{
		conn:   conn,
		window: window,
		atoms: atoms{
			atom:      internAtom(conn, "ATOM"),
			clipboard: internAtom(conn, "CLIPBOARD"),
			string:    internAtom(conn, "STRING"),
			targets:   internAtom(conn, "TARGETS"),
			text:      internAtom(conn, "TEXT"),
			utf8:      internAtom(conn, "UTF8_STRING"),
			wmClass:   internAtom(conn, "WM_CLASS"),
			wmName:    internAtom(conn, "WM_NAME"),
			netWmName: internAtom(conn, "_NET_WM_NAME"),
		},
		logger: logger,
		done:   make(chan struct{}),
	}

	go writer.eventLoop()
	return writer, nil
}

// OfferEmbed replaces the current clipboard offer with a lazy embed.
func (w *X11Writer) OfferEmbed(filename string, onPaste func() error) error {
	w.mu.Lock()
	w.current = &offer{
		filename:  filename,
		text:      fmt.Sprintf("![[%s]]", filename),
		onPaste:   onPaste,
		offeredAt: time.Now(),
	}
	w.mu.Unlock()

	if err := xproto.SetSelectionOwnerChecked(
		w.conn,
		w.window,
		w.atoms.clipboard,
		xproto.TimeCurrentTime,
	).Check(); err != nil {
		return fmt.Errorf("set clipboard owner: %w", err)
	}

	reply, err := xproto.GetSelectionOwner(w.conn, w.atoms.clipboard).Reply()
	if err != nil {
		return fmt.Errorf("confirm clipboard owner: %w", err)
	}
	if reply.Owner != w.window {
		return fmt.Errorf("clipboard owner mismatch: got %d want %d", reply.Owner, w.window)
	}

	return nil
}

// Close releases clipboard ownership and tears down the X11 connection.
func (w *X11Writer) Close() error {
	select {
	case <-w.done:
		return nil
	default:
	}

	w.mu.Lock()
	w.current = nil
	w.mu.Unlock()

	_ = xproto.SetSelectionOwnerChecked(w.conn, xproto.WindowNone, w.atoms.clipboard, xproto.TimeCurrentTime).Check()
	_ = xproto.DestroyWindowChecked(w.conn, w.window).Check()
	close(w.done)
	w.conn.Close()
	return nil
}

func (w *X11Writer) eventLoop() {
	for {
		select {
		case <-w.done:
			return
		default:
		}

		event, err := w.conn.WaitForEvent()
		if err != nil || event == nil {
			select {
			case <-w.done:
				return
			default:
				continue
			}
		}

		switch e := event.(type) {
		case xproto.SelectionClearEvent:
			w.handleSelectionClear(e)
		case xproto.SelectionRequestEvent:
			w.handleSelectionRequest(e)
		}
	}
}

func (w *X11Writer) handleSelectionClear(event xproto.SelectionClearEvent) {
	if event.Selection != w.atoms.clipboard {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != nil && !w.current.saved {
		w.logger.Printf("Discarded pending attachment: %s", w.current.filename)
	}
	w.current = nil
}

func (w *X11Writer) handleSelectionRequest(event xproto.SelectionRequestEvent) {
	w.mu.Lock()
	current := w.current
	w.mu.Unlock()

	property := event.Property
	if property == xproto.AtomNone {
		property = event.Target
	}

	if current == nil {
		w.notifySelection(event, xproto.AtomNone)
		return
	}

	switch event.Target {
	case w.atoms.targets:
		buf := make([]byte, 4*4)
		xgb.Put32(buf[0:], uint32(w.atoms.targets))
		xgb.Put32(buf[4:], uint32(w.atoms.utf8))
		xgb.Put32(buf[8:], uint32(w.atoms.text))
		xgb.Put32(buf[12:], uint32(w.atoms.string))

		if err := xproto.ChangePropertyChecked(
			w.conn,
			xproto.PropModeReplace,
			event.Requestor,
			property,
			w.atoms.atom,
			32,
			4,
			buf,
		).Check(); err != nil {
			w.logger.Printf("Failed to serve TARGETS: %v", err)
			w.notifySelection(event, xproto.AtomNone)
			return
		}

		w.notifySelection(event, property)
	case w.atoms.utf8, w.atoms.text, w.atoms.string:
		requestor := w.lookupRequestor(event.Requestor)
		if current.saved {
			// Already persisted; just serve the embed text again.
		} else if age := time.Since(current.offeredAt); age < initialProbeWindow {
			w.logger.Printf("Deferring initial clipboard probe from %s (%s after copy)", requestor, age.Round(10*time.Millisecond))
		} else if shouldSaveForRequestor(requestor, age) {
			w.logger.Printf("Saving attachment for paste request from %s", requestor)
			if err := w.ensureSaved(current); err != nil {
				w.logger.Printf("Failed to save attachment on paste: %v", err)
				w.notifySelection(event, xproto.AtomNone)
				return
			}
		} else {
			w.logger.Printf("Serving unsaved embed to non-paste requestor %s", requestor)
		}

		if err := xproto.ChangePropertyChecked(
			w.conn,
			xproto.PropModeReplace,
			event.Requestor,
			property,
			event.Target,
			8,
			uint32(len(current.text)),
			[]byte(current.text),
		).Check(); err != nil {
			w.logger.Printf("Failed to serve embed text: %v", err)
			w.notifySelection(event, xproto.AtomNone)
			return
		}

		w.notifySelection(event, property)
	default:
		w.notifySelection(event, xproto.AtomNone)
	}
}

func (w *X11Writer) lookupRequestor(window xproto.Window) requestorInfo {
	return requestorInfo{
		class: w.getWindowProperty(window, w.atoms.wmClass),
		name:  firstNonEmpty(
			w.getWindowProperty(window, w.atoms.netWmName),
			w.getWindowProperty(window, w.atoms.wmName),
		),
	}
}

func (w *X11Writer) getWindowProperty(window xproto.Window, property xproto.Atom) string {
	reply, err := xproto.GetProperty(w.conn, false, window, property, 0, 0, 1024).Reply()
	if err != nil || reply == nil || len(reply.Value) == 0 {
		return ""
	}

	if property == w.atoms.wmClass {
		return normalizeNullSeparated(reply.Value)
	}

	return string(bytes.TrimRight(reply.Value, "\x00"))
}

func (w *X11Writer) ensureSaved(current *offer) error {
	w.mu.Lock()
	if current != w.current {
		w.mu.Unlock()
		return fmt.Errorf("clipboard offer changed before paste")
	}
	if current.saved {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	if err := current.onPaste(); err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if current != w.current {
		return fmt.Errorf("clipboard offer changed while saving")
	}
	current.saved = true
	return nil
}

func (w *X11Writer) notifySelection(event xproto.SelectionRequestEvent, property xproto.Atom) {
	notify := xproto.SelectionNotifyEvent{
		Time:      event.Time,
		Requestor: event.Requestor,
		Selection: event.Selection,
		Target:    event.Target,
		Property:  property,
	}

	if err := xproto.SendEventChecked(w.conn, false, event.Requestor, 0, string(notify.Bytes())).Check(); err != nil {
		w.logger.Printf("Failed to send SelectionNotify: %v", err)
	}
}

func shouldSaveForRequestor(requestor requestorInfo, age time.Duration) bool {
	if requestor.isUnknown() {
		return age >= unknownRequestorWindow
	}

	combined := strings.ToLower(requestor.class + " " + requestor.name)
	for _, marker := range []string{
		"obsidian",
		"obsidian.exe",
		"com.obsidian",
		"md.obsidian",
		"chromium",
		"electron",
		"chrome",
	} {
		if strings.Contains(combined, marker) {
			return true
		}
	}

	return false
}

func normalizeNullSeparated(value []byte) string {
	parts := bytes.Split(bytes.TrimRight(value, "\x00"), []byte{0})
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		fields = append(fields, string(part))
	}

	return strings.Join(fields, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func (r requestorInfo) String() string {
	switch {
	case r.class != "" && r.name != "":
		return fmt.Sprintf("%s (%s)", r.class, r.name)
	case r.class != "":
		return r.class
	case r.name != "":
		return r.name
	default:
		return "unknown requestor"
	}
}

func (r requestorInfo) isUnknown() bool {
	return r.class == "" && r.name == ""
}

func internAtom(conn *xgb.Conn, name string) xproto.Atom {
	reply, err := xproto.InternAtom(conn, true, uint16(len(name)), name).Reply()
	if err != nil {
		panic(fmt.Errorf("intern atom %q: %w", name, err))
	}
	return reply.Atom
}
