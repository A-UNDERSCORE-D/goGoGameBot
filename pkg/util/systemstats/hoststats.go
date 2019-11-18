package systemstats

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/version"
)

func getSystemUsageStats() string {
	out := strings.Builder{}
	out.WriteString("CPU Load: ")
	if h, err := cpu.Percent(time.Millisecond*50, false); err != nil {
		out.WriteString("Error ")
	} else {
		out.WriteString(fmt.Sprintf("%.2f%% ", h[0]))
	}
	out.WriteString("Memory Usage: ")
	if m, err := mem.VirtualMemory(); err != nil {
		out.WriteString("Error ")
	} else {
		out.WriteString(fmt.Sprintf("%s/%s (%.2f%%)", humanize.IBytes(m.Used), humanize.IBytes(m.Total), m.UsedPercent))
	}
	return out.String()
}

func getBotUsageStats() string {
	out := strings.Builder{}
	out.WriteString(fmt.Sprintf("Version: %s ", version.Version))
	memstats := new(runtime.MemStats)
	runtime.ReadMemStats(memstats)
	out.WriteString(fmt.Sprintf(
		"Memory Usage: %s",
		humanize.IBytes(memstats.Sys),
	))
	return out.String()
}

func getGoStats() string {
	out := strings.Builder{}
	out.WriteString("Goroutines: ")
	out.WriteString(strconv.Itoa(runtime.NumGoroutine()))
	out.WriteString(" Version: ")
	out.WriteString(runtime.Version())
	return out.String()
}

// GetStats returns a string containing statistics of the currently running bot, and the system as a whole
func GetStats() string {
	return fmt.Sprintf("Bot: %s System: %s Go: %s", getBotUsageStats(), getSystemUsageStats(), getGoStats())
}
