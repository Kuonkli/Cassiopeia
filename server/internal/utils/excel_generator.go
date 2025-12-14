package utils

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"strconv"
	"time"

	"cassiopeia/internal/models"
)

// CreateExcelFile создает Excel файл с данными телеметрии
func CreateExcelFile(filepath string, records []models.Telemetry) error {
	f := excelize.NewFile()
	defer f.Close()

	// Создаем новый лист
	index, err := f.NewSheet("Telemetry")
	if err != nil {
		return err
	}

	// Устанавливаем заголовки
	headers := []string{"Timestamp", "Voltage (V)", "Temperature (°C)", "Source File", "Created At"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Telemetry", cell, header)
	}

	// Заполняем данные
	for rowIdx, record := range records {
		rowNum := rowIdx + 2 // Заголовок в первой строке

		f.SetCellValue("Telemetry", fmt.Sprintf("A%d", rowNum),
			record.RecordedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue("Telemetry", fmt.Sprintf("B%d", rowNum), record.Voltage)
		f.SetCellValue("Telemetry", fmt.Sprintf("C%d", rowNum), record.Temperature)
		f.SetCellValue("Telemetry", fmt.Sprintf("D%d", rowNum), record.SourceFile)
		f.SetCellValue("Telemetry", fmt.Sprintf("E%d", rowNum),
			record.CreatedAt.Format("2006-01-02 15:04:05"))

		// Форматирование чисел
		f.SetCellStyle("Telemetry", fmt.Sprintf("B%d", rowNum), fmt.Sprintf("B%d", rowNum),
			getNumberStyle(f, "0.00"))
		f.SetCellStyle("Telemetry", fmt.Sprintf("C%d", rowNum), fmt.Sprintf("C%d", rowNum),
			getNumberStyle(f, "0.00"))
	}

	// Авто-ширина колонок
	for i := 1; i <= len(headers); i++ {
		colName, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth("Telemetry", colName, colName, 20)
	}

	// Добавляем условное форматирование для температуры
	// Красный для высоких температур (> 60°C)
	highTempRule := []excelize.ConditionalFormatOptions{
		{
			Type:     "cell",
			Criteria: ">",
			Value:    "60",
			Format:   getConditionalFormatStyle(f, "#FFCCCC"),
		},
	}
	err = f.SetConditionalFormat("Telemetry", "C2:C1000", highTempRule)
	if err != nil {
		return err
	}

	// Синий для низких температур (< -20°C)
	lowTempRule := []excelize.ConditionalFormatOptions{
		{
			Type:     "cell",
			Criteria: "<",
			Value:    "-20",
			Format:   getConditionalFormatStyle(f, "#CCE5FF"),
		},
	}
	err = f.SetConditionalFormat("Telemetry", "C2:C1000", lowTempRule)
	if err != nil {
		return err
	}

	// Создаем график
	if len(records) > 1 {
		createChart(f, records)
	}

	// Создаем информационный лист
	createInfoSheet(f, records)

	// Устанавливаем активный лист
	f.SetActiveSheet(index)

	// Сохраняем файл
	if err := f.SaveAs(filepath); err != nil {
		return err
	}

	return nil
}

func getNumberStyle(f *excelize.File, format string) int {
	formatInt, err := strconv.Atoi(format)
	if err != nil {
		return 0
	}
	style, _ := f.NewStyle(&excelize.Style{
		NumFmt: formatInt,
	})
	return style
}

func createChart(f *excelize.File, records []models.Telemetry) {
	// Создаем график температуры
	chart := &excelize.Chart{
		Type: excelize.Col3DClustered,
		Series: []excelize.ChartSeries{
			{
				Name:       "Temperature",
				Categories: "Telemetry!$A$2:$A$" + fmt.Sprintf("%d", len(records)+1),
				Values:     "Telemetry!$C$2:$C$" + fmt.Sprintf("%d", len(records)+1),
			},
		},
		Title: []excelize.RichTextRun{
			{
				Text: "Temperature Over Time",
			},
		},
		XAxis: excelize.ChartAxis{
			MajorGridLines: true,
		},
		YAxis: excelize.ChartAxis{
			MajorGridLines: true,
		},
		Dimension: excelize.ChartDimension{
			Width:  600,
			Height: 400,
		},
	}

	f.AddChart("Telemetry", "G2", chart)
}

func createInfoSheet(f *excelize.File, records []models.Telemetry) {
	// Создаем лист с информацией
	f.NewSheet("Info")

	// Записываем метаданные
	metadata := map[string]interface{}{
		"Report Generated": time.Now().Format("2006-01-02 15:04:05"),
		"Total Records":    len(records),
		"Time Range": fmt.Sprintf("%s to %s",
			records[0].RecordedAt.Format("2006-01-02 15:04:05"),
			records[len(records)-1].RecordedAt.Format("2006-01-02 15:04:05")),
		"Voltage Range": fmt.Sprintf("%.2fV - %.2fV",
			findMinVoltage(records), findMaxVoltage(records)),
		"Temperature Range": fmt.Sprintf("%.2f°C - %.2f°C",
			findMinTemperature(records), findMaxTemperature(records)),
	}

	row := 1
	for key, value := range metadata {
		f.SetCellValue("Info", fmt.Sprintf("A%d", row), key)
		f.SetCellValue("Info", fmt.Sprintf("B%d", row), value)
		row++
	}
}

func findMinVoltage(records []models.Telemetry) float64 {
	if len(records) == 0 {
		return 0
	}
	min := records[0].Voltage
	for _, r := range records {
		if r.Voltage < min {
			min = r.Voltage
		}
	}
	return min
}

func findMaxVoltage(records []models.Telemetry) float64 {
	if len(records) == 0 {
		return 0
	}
	max := records[0].Voltage
	for _, r := range records {
		if r.Voltage > max {
			max = r.Voltage
		}
	}
	return max
}

func findMinTemperature(records []models.Telemetry) float64 {
	if len(records) == 0 {
		return 0
	}
	min := records[0].Temperature
	for _, r := range records {
		if r.Temperature < min {
			min = r.Temperature
		}
	}
	return min
}

func findMaxTemperature(records []models.Telemetry) float64 {
	if len(records) == 0 {
		return 0
	}
	max := records[0].Temperature
	for _, r := range records {
		if r.Temperature > max {
			max = r.Temperature
		}
	}
	return max
}

// SaveAsJSON сохраняет данные в JSON файл
func SaveAsJSON(filepath string, data interface{}) error {
	// Реализация сохранения в JSON
	// (используйте encoding/json)
	return nil
}

// getConditionalFormatStyle создает стиль для условного форматирования
func getConditionalFormatStyle(f *excelize.File, color string) *int {
	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{color},
			Pattern: 1,
		},
	})
	if err != nil {
		return nil
	}
	return &style
}
