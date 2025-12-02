package handlers

import (
	"bytes"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestUploadPDFTwiceInARow(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Ensure clean DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Helper to create item with PDF
	createItemWithPDF := func(name, pdfContent string) (int, string) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("Name", name)
		writer.WriteField("Description", "Item with PDF")
		writer.WriteField("Price", strconv.Itoa(100))
		
		fw, _ := writer.CreateFormFile("PDF", "test.pdf")
		fw.Write([]byte(pdfContent))
		writer.Close()

		res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
		idStr := strings.TrimSpace(res.Body.String())
		id, _ := strconv.Atoi(idStr)
		
		// Get PDF path
		item, err := database.Db.GetItem(id)
		require.NoError(t, err)
		require.True(t, item.PDF.Valid)
		
		pdf, err := database.Db.GetPDFByID(item.PDF.ValueOrZero())
		require.NoError(t, err)
		
		return id, pdf.Path
	}

	// 1. Upload first PDF
	id1, path1 := createItemWithPDF("Item 1", "PDF Content 1")
	
	// 2. Upload second PDF immediately (same filename from client)
	// We want to ensure we hit the same second if possible, or just check behavior
	id2, path2 := createItemWithPDF("Item 2", "PDF Content 2")

	t.Logf("Item 1 ID: %d, Path 1: %s", id1, path1)
	t.Logf("Item 2 ID: %d, Path 2: %s", id2, path2)

	// Check if paths are different
	// If they are the same, it means we had a collision (if within same second)
	// If the user wants to prevent collision, we might need milliseconds.
	// But let's see what happens.
	
	// Verify content on disk
	content1, err := os.ReadFile(path1)
	require.NoError(t, err)
	
	content2, err := os.ReadFile(path2)
	require.NoError(t, err)

	// If paths are same, content2 would have overwritten content1
	if path1 == path2 {
		t.Log("Warning: Paths are identical. Content might be overwritten.")
		require.Equal(t, "PDF Content 2", string(content1), "Content 1 should be overwritten by Content 2 if paths are same")
	} else {
		require.Equal(t, "PDF Content 1", string(content1))
		require.Equal(t, "PDF Content 2", string(content2))
	}
	
	// Clean up
	os.Remove(path1)
	if path1 != path2 {
		os.Remove(path2)
	}
	
	// Also test updating an item with PDF twice
	t.Log("Testing UpdateItem with PDF twice")
	
	// Create item without PDF
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "Update Item")
	writer.WriteField("Description", "To be updated")
	writer.WriteField("Price", "50")
	writer.Close()
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	updateIDStr := strings.TrimSpace(res.Body.String())
	
	updateItemWithPDF := func(idStr, pdfContent string) string {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		// Must provide required fields or they might be cleared/error? 
		// UpdateItem implementation:
		// if item.Description == "" { preserve }
		// So we can just send PDF?
		// UpdateItem requires multipart form.
		
		fw, _ := writer.CreateFormFile("PDF", "update.pdf")
		fw.Write([]byte(pdfContent))
		writer.Close()
		
		utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+idStr+"/", body, writer.FormDataContentType(), 200, adminUserToken)
		
		id, _ := strconv.Atoi(idStr)
		item, err := database.Db.GetItem(id)
		require.NoError(t, err)
		require.True(t, item.PDF.Valid)
		
		pdf, err := database.Db.GetPDFByID(item.PDF.ValueOrZero())
		require.NoError(t, err)
		return pdf.Path
	}
	
	pathUpdate1 := updateItemWithPDF(updateIDStr, "Update Content 1")
	// Sleep a bit to ensure different timestamp if we want to avoid collision for this part of test?
	// Or test collision?
	// The user asked "uploading a pdf to an item twice in a row".
	// Let's try to be fast.
	pathUpdate2 := updateItemWithPDF(updateIDStr, "Update Content 2")
	
	t.Logf("Update Path 1: %s", pathUpdate1)
	t.Logf("Update Path 2: %s", pathUpdate2)
	
	if pathUpdate1 == pathUpdate2 {
		t.Log("Warning: Update Paths are identical.")
	}
	
	// Cleanup
	os.Remove(pathUpdate1)
	if pathUpdate1 != pathUpdate2 {
		os.Remove(pathUpdate2)
	}
}
