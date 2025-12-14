package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"cassiopeia/internal/models"
	"cassiopeia/internal/repository"
	"cassiopeia/internal/utils"
)

type TelemetryService interface {
	GenerateTelemetry(ctx context.Context) (*TelemetryBatch, error)
	GenerateTelemetryCSV(ctx context.Context) (string, error)
	GenerateTelemetryExcel(ctx context.Context) (string, error)
	GetTelemetryHistory(ctx context.Context, from, to time.Time) ([]models.Telemetry, error)
	ExportTelemetry(ctx context.Context, format string, from, to time.Time) (string, error)
}

type telemetryService struct {
	repo      repository.TelemetryRepository
	outputDir string
}

type TelemetryBatch struct {
	Filename    string             `json:"filename"`
	Records     int                `json:"records"`
	GeneratedAt time.Time          `json:"generated_at"`
	Data        []models.Telemetry `json:"data,omitempty"`
}

func NewTelemetryService(repo repository.TelemetryRepository, outputDir string) TelemetryService {
	if outputDir == "" {
		outputDir = "/data/telemetry"
	}

	// Создаем директорию если не существует
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Failed to create telemetry directory: %v", err)
	}

	return &telemetryService{
		repo:      repo,
		outputDir: outputDir,
	}
}

func (s *telemetryService) GenerateTelemetry(ctx context.Context) (*TelemetryBatch, error) {
	log.Println("Generating telemetry data...")

	// Генерируем имя файла с timestamp
	timestamp := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("telemetry_%s.csv", timestamp)
	filepath := filepath.Join(s.outputDir, filename)

	// Генерируем тестовые данные
	records := s.generateSampleData(100) // 100 записей

	// Сохраняем в CSV
	if err := s.saveToCSV(filepath, records); err != nil {
		return nil, fmt.Errorf("failed to save CSV: %w", err)
	}

	// Сохраняем в БД
	for _, record := range records {
		if err := s.repo.Create(ctx, &record); err != nil {
			log.Printf("Failed to save telemetry record to DB: %v", err)
		}
	}

	log.Printf("Telemetry generated: %s (%d records)", filename, len(records))

	return &TelemetryBatch{
		Filename:    filename,
		Records:     len(records),
		GeneratedAt: time.Now().UTC(),
		Data:        records,
	}, nil
}

func (s *telemetryService) generateSampleData(count int) []models.Telemetry {
	var records []models.Telemetry
	rand.Seed(time.Now().UnixNano())

	startTime := time.Now().UTC().Add(-time.Duration(count) * time.Second)

	for i := 0; i < count; i++ {
		recordTime := startTime.Add(time.Duration(i) * time.Second)

		record := models.Telemetry{
			RecordedAt:  recordTime,
			Voltage:     randFloat(3.2, 12.6),
			Temperature: randFloat(-50.0, 80.0),
			SourceFile:  fmt.Sprintf("telemetry_%s.csv", recordTime.Format("20060102_150405")),
			CreatedAt:   time.Now().UTC(),
		}

		records = append(records, record)
	}

	return records
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (s *telemetryService) saveToCSV(filepath string, records []models.Telemetry) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовок
	header := []string{"recorded_at", "voltage", "temperature", "source_file"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Записываем данные
	for _, record := range records {
		row := []string{
			record.RecordedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", record.Voltage),
			fmt.Sprintf("%.2f", record.Temperature),
			record.SourceFile,
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (s *telemetryService) GenerateTelemetryCSV(ctx context.Context) (string, error) {
	batch, err := s.GenerateTelemetry(ctx)
	if err != nil {
		return "", err
	}

	filepath := filepath.Join(s.outputDir, batch.Filename)
	return filepath, nil
}

func (s *telemetryService) GenerateTelemetryExcel(ctx context.Context) (string, error) {
	// Генерируем данные
	batch, err := s.GenerateTelemetry(ctx)
	if err != nil {
		return "", err
	}

	// Создаем Excel файл
	excelFilename := fmt.Sprintf("telemetry_%s.xlsx",
		time.Now().UTC().Format("20060102_150405"))
	excelPath := filepath.Join(s.outputDir, excelFilename)

	// Используем утилиту для создания Excel
	if err := utils.CreateExcelFile(excelPath, batch.Data); err != nil {
		return "", fmt.Errorf("failed to create Excel file: %w", err)
	}

	log.Printf("Excel file generated: %s", excelFilename)
	return excelPath, nil
}

func (s *telemetryService) GetTelemetryHistory(ctx context.Context, from, to time.Time) ([]models.Telemetry, error) {
	if from.IsZero() {
		from = time.Now().UTC().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}

	// Ограничиваем диапазон 30 днями
	maxRange := 30 * 24 * time.Hour
	if to.Sub(from) > maxRange {
		from = to.Add(-maxRange)
	}

	return s.repo.GetByDateRange(ctx, from, to)
}

func (s *telemetryService) ExportTelemetry(ctx context.Context, format string, from, to time.Time) (string, error) {
	// Получаем данные
	records, err := s.GetTelemetryHistory(ctx, from, to)
	if err != nil {
		return "", fmt.Errorf("failed to get telemetry data: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("no data found for the specified range")
	}

	timestamp := time.Now().UTC().Format("20060102_150405")

	switch format {
	case "csv":
		filename := fmt.Sprintf("telemetry_export_%s.csv", timestamp)
		filepath := filepath.Join(s.outputDir, filename)

		if err := s.saveToCSV(filepath, records); err != nil {
			return "", err
		}

		return filepath, nil

	case "excel", "xlsx":
		filename := fmt.Sprintf("telemetry_export_%s.xlsx", timestamp)
		filepath := filepath.Join(s.outputDir, filename)

		if err := utils.CreateExcelFile(filepath, records); err != nil {
			return "", err
		}

		return filepath, nil

	case "json":
		filename := fmt.Sprintf("telemetry_export_%s.json", timestamp)
		filepath := filepath.Join(s.outputDir, filename)

		if err := utils.SaveAsJSON(filepath, records); err != nil {
			return "", err
		}

		return filepath, nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}
