// Command media-migrate inventories images by default; -apply copies them to COS.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/cosstore"
	"time"
)

func run() error {
	path := flag.String("conf", conf.DefaultPath, "private YAML configuration")
	apply := flag.Bool("apply", false, "copy local images to COS; keep local files and IDs")
	flag.Parse()
	c, err := conf.LoadFile(*path)
	if err != nil {
		return err
	}
	d, closeDB, err := data.NewData(c)
	if err != nil {
		return err
	}
	defer closeDB()
	var rows []struct {
		ID   int64
		Name string
		Size int64
	}
	if err := d.DB.Table("media").Select("id,name,size").Where("storage_provider='local' AND media_type='image' AND available AND deleted_at IS NULL").Order("id").Find(&rows).Error; err != nil {
		return err
	}
	var legacy int64
	if err := d.DB.Table("media").Where("media_type!='image' AND deleted_at IS NULL").Count(&legacy).Error; err != nil {
		return err
	}
	fmt.Printf("Local images: %d; retained non-image records: %d\n", len(rows), legacy)
	if !*apply {
		for _, row := range rows {
			fmt.Printf("id=%d size=%d name=%q\n", row.ID, row.Size, row.Name)
		}
		return nil
	}
	if !c.Data.Storage.COS.Enabled {
		return fmt.Errorf("configure and enable COS before applying migration")
	}
	r := media.NewRepo(d)
	store := cosstore.New(c.Data.Storage.COS)
	for _, row := range rows {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		err := r.MigrateLocal(ctx, c.Data.Storage, row.ID, store)
		cancel()
		if err != nil {
			return fmt.Errorf("image %d migration failed: %w", row.ID, err)
		}
		fmt.Printf("image %d migrated and verified\n", row.ID)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
