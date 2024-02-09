package main

import (
	"log"

	"github.com/3c7/aen"
)

// listRecipients lists all recipients or remove a recipient with a specific alias
func listRecipients(pathFlag, aliasFlag string) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if aliasFlag != "" {
		err = db.RemoveRecipientByAlias(aliasFlag)
		if err != nil {
			log.Fatalf("Error removing recipient %s: %v", aliasFlag, err)
		}
		return
	}

	recipients, err := db.GetRecipients()
	if err != nil {
		log.Fatalf("Error loading recipients: %v", err)
	}
	if len(recipients) == 0 {
		// Should not really be the case, but anyway...
		log.Println("Recipient list is empty.")
	} else {
		log.Printf("| %-20s | %-62s |", "Alias", "Public Key")
		for _, r := range recipients {
			log.Printf("| %-20s | %-62s |", r.Alias, r.Publickey)
		}
	}
}
