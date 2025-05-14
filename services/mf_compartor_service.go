package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"stockbackend/types"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

type MFCompartorServiceI interface {
	ParseXLSXFiles(ctx *gin.Context, files <-chan string, sentryCtx context.Context) error
}

type mFCompartorfileService struct{}

var MFCompartorService MFCompartorServiceI = &mFCompartorfileService{}

func (fs *mFCompartorfileService) ParseXLSXFiles(ctx *gin.Context, files <-chan string, sentryCtx context.Context) error {
	defer sentry.Recover()
	span := sentry.StartSpan(sentryCtx, "[DAO] ParseXLSXFile")
	defer span.Finish()

	var mfData []types.MFInstrument

	for filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			sentry.CaptureException(err)
			zap.L().Error("Error opening file", zap.String("filePath", filePath), zap.Error(err))
			if err := os.Remove(filePath); err != nil {
				zap.L().Error("Error removing file", zap.String("filePath", filePath), zap.Error(err))
			} else {
				zap.L().Info("File removed successfully", zap.String("filePath", filePath))
			}
			continue
		}
		defer file.Close()

		// Create a new reader from the uploaded file
		if _, err := file.Seek(0, 0); err != nil {
			zap.L().Error("Error seeking file", zap.String("filePath", filePath), zap.Error(err))
			sentry.CaptureException(err)
			return err
		}

		f, err := excelize.OpenReader(file)
		if err != nil {
			sentry.CaptureException(err)
			zap.L().Error("Error parsing XLSX file", zap.String("filePath", filePath), zap.Error(err))
			if err := os.Remove(filePath); err != nil {
				zap.L().Error("Error removing file", zap.String("filePath", filePath), zap.Error(err))
			} else {
				zap.L().Info("File removed successfully", zap.String("filePath", filePath))
			}
			continue
		}
		defer f.Close()

		sheetList := f.GetSheetList()
		for _, sheet := range sheetList {
			zap.L().Info("Processing file", zap.String("filePath", filePath), zap.String("sheet", sheet))
			// Get all the rows in the sheet
			rows, err := f.GetRows(sheet)
			if err != nil {
				sentry.CaptureException(err)
				zap.L().Error("Error reading rows from sheet", zap.String("sheet", sheet), zap.Error(err))
				continue
			}
			mfSummary := CallGeminiAPI(rows)
			if len(mfSummary.FundData) > 0 {
				mfData = append(mfData, types.MFInstrument{
					Instruments: mfSummary.FundData,
					Name:        mfSummary.MutualFundName,
				})
			}
		}
		if err := os.Remove(filePath); err != nil {
			sentry.CaptureException(err)
			zap.L().Error("Error removing file", zap.String("filePath", filePath), zap.Error(err))
		} else {
			zap.L().Info("File removed successfully", zap.String("filePath", filePath))
		}

	}
	mutualFund1WeightPercentage, mutualFund2WeightPercentage := calculateMutualFundOverlap(mfData[0].Instruments, mfData[1].Instruments)
	mutualFund1Percentage, mutualFund2Percentage, commonStocks := calculateOverlapPercentage(mfData[0].Instruments, mfData[1].Instruments)

	overlapMutualFund := types.OverlapMutualFund{
		Fund1Percentage:       fmt.Sprintf("%.2f", mutualFund1Percentage),
		Fund2Percentage:       fmt.Sprintf("%.2f", mutualFund2Percentage),
		Fund1PercentageWeight: fmt.Sprintf("%.2f", mutualFund1WeightPercentage),
		Fund2PercentageWeight: fmt.Sprintf("%.2f", mutualFund2WeightPercentage),
		CommonStocks:          commonStocks,
	}
	jsonData, err := json.Marshal(overlapMutualFund)
	if err != nil {
		sentry.CaptureException(err)
		zap.L().Error("Error marshalling data to JSON", zap.Error(err))
		return err
	}
	_, err = ctx.Writer.Write(append(jsonData, '\n'))

	if err != nil {
		sentry.CaptureException(err)
		zap.L().Error("Error writing data", zap.Error(err))

	}
	ctx.Writer.Flush()

	return nil
}
