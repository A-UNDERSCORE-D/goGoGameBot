package systemstats

import (
    "fmt"
    "runtime"
    "strings"
    "time"

    "github.com/dustin/go-humanize"
    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/mem"
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
    memstats := new(runtime.MemStats)
    runtime.ReadMemStats(memstats)
    out.WriteString(fmt.Sprintf(
        "Memory Usage: %s",
        humanize.IBytes(memstats.Sys),
    ))
    return out.String()
}

func GetStats() string {
    return fmt.Sprintf("Bot: %s System: %s", getBotUsageStats(), getSystemUsageStats())
}
