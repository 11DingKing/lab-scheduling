package domain

import "time"

type WindowDTO struct{ Start, End string }

func WindowToDTO(w TimeWindow) WindowDTO {
	return WindowDTO{Start: w.Start.Format(time.RFC3339Nano), End: w.End.Format(time.RFC3339Nano)}
}
func WindowFromDTO(v WindowDTO) (TimeWindow, error) {
	start, err := time.Parse(time.RFC3339Nano, v.Start)
	if err != nil {
		return TimeWindow{}, err
	}
	end, err := time.Parse(time.RFC3339Nano, v.End)
	if err != nil {
		return TimeWindow{}, err
	}
	return NewWindow(start, end)
}
func StatusIsActive(s ReservationStatus) bool {
	return s == ReservationHeld || s == ReservationCheckedOut || s == ReservationSuspended
}
func StatusIsTerminal(s ReservationStatus) bool {
	return s == ReservationReturned || s == ReservationCancelled || s == ReservationNoShow
}
