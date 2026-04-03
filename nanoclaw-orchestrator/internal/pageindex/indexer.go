package pageindex

import (
	"database/sql"
	"fmt"
	"nanoclaw-orchestrator/internal"
)

// Node represents a single structural block in the PageIndex tree
type Node struct {
	ID           int
	DocumentID   string
	ParentNodeID int
	NodeID       string
	Title        string
	Summary      string
	Content      string
}

type Indexer struct {
	db *internal.Database
}

func NewIndexer(db *internal.Database) *Indexer {
	return &Indexer{db: db}
}

// AddNode adds a new node to the tree for a specific document.
func (idx *Indexer) AddNode(documentID string, parentNodeID int, nodeID, title, summary, content string) (int64, error) {
	query := `INSERT INTO pageindex_nodes (document_id, parent_node_id, node_id, title, summary, content) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	res, err := idx.db.Conn.Exec(query, documentID, parentNodeID, nodeID, title, summary, content)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetTableOfContents retrieves the tree structure without full content,
// returning formatted markdown suitable for prompting MiniMax.
func (idx *Indexer) GetTableOfContents(documentID string) (string, error) {
	query := `SELECT id, parent_node_id, node_id, title, summary 
			  FROM pageindex_nodes 
			  WHERE document_id = ? 
			  ORDER BY id ASC`
	rows, err := idx.db.Conn.Query(query, documentID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var toc string
	toc += fmt.Sprintf("Table of Contents for Document: %s\n", documentID)
	toc += "================================================\n\n"

	for rows.Next() {
		var id, parentNodeID int
		var nodeID, title, summary sql.NullString
		
		if err := rows.Scan(&id, &parentNodeID, &nodeID, &title, &summary); err != nil {
			return "", err
		}
		
		// Indent based on whether it is a root node (0) or child
		indent := ""
		if parentNodeID != 0 {
			indent = "    - "
		} else {
			indent = "- "
		}
		
		toc += fmt.Sprintf("%s[%s] %s: %s\n", indent, nodeID.String, title.String, summary.String)
	}
	return toc, nil
}

// FetchNodeContent simulates the Reasoning step where the LLM requests a specific
// node ID after reading the Table of Contents.
func (idx *Indexer) FetchNodeContent(documentID, nodeID string) (string, error) {
	query := `SELECT title, content FROM pageindex_nodes WHERE document_id = ? AND node_id = ?`
	var title, content string
	err := idx.db.Conn.QueryRow(query, documentID, nodeID).Scan(&title, &content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("node %s not found in document %s", nodeID, documentID)
		}
		return "", err
	}
	
	return fmt.Sprintf("=== Node: %s | Title: %s ===\n\n%s\n", nodeID, title, content), nil
}
