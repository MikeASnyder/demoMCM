package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bep/debounce"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	helmlib "github.com/rancher/rancher/pkg/helm"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/rancher/wrangler/pkg/ticker"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CacheCleaner struct {
	catalogClient        v3.CatalogInterface
	projectCatalogClient v3.ProjectCatalogInterface
	clusterCatalogClient v3.ClusterCatalogInterface
	debounce             func(func())
}

func Register(ctx context.Context, context *config.ScaledContext) {
	cleaner := &CacheCleaner{
		catalogClient:        context.Management.Catalogs(""),
		projectCatalogClient: context.Management.ProjectCatalogs(""),
		clusterCatalogClient: context.Management.ClusterCatalogs(""),
		debounce:             debounce.New(time.Minute),
	}
	go cleaner.runPeriodicCatalogCacheCleaner(ctx, time.Hour)

	context.Management.Catalogs("").Controller().AddHandler(ctx, "catalogCache", cleaner.destroyCatalogSync)
	context.Management.ClusterCatalogs("").Controller().AddHandler(ctx, "clusterCatalogCache", cleaner.destroyClusterCatalogSync)
	context.Management.ProjectCatalogs("").Controller().AddHandler(ctx, "projectCatalogCache", cleaner.destroyProjectCatalogSync)
}

func (c *CacheCleaner) runPeriodicCatalogCacheCleaner(ctx context.Context, interval time.Duration) {
	c.GoCleanupLogError()
	for range ticker.Context(ctx, interval) {
		c.GoCleanupLogError()
	}
}

func (c *CacheCleaner) destroyCatalogSync(key string, obj *v3.Catalog) (runtime.Object, error) {
	c.debounce(c.GoCleanupLogError)
	return nil, nil
}

func (c *CacheCleaner) destroyClusterCatalogSync(key string, obj *v3.ClusterCatalog) (runtime.Object, error) {
	c.debounce(c.GoCleanupLogError)
	return nil, nil
}

func (c *CacheCleaner) destroyProjectCatalogSync(key string, obj *v3.ProjectCatalog) (runtime.Object, error) {
	c.debounce(c.GoCleanupLogError)
	return nil, nil
}

func (c *CacheCleaner) GoCleanupLogError() {
	go func() {
		if err := c.Cleanup(); err != nil {
			logrus.Errorf("Catalog-cache cleanup error: %s", err)
		}
	}()
}

func contains(catalogs []*v3.Catalog, c *v3.Catalog) bool {
	for _, catalog := range catalogs {
		if catalog.Name == c.Name {
			return true
		}
	}
	return false
}

func (c *CacheCleaner) cleanUpCatalogs(targetCatalogs []*v3.Catalog) error {
	logrus.Debug("Catalog-cache running cleanUpCatalogs")
	catalogCacheFiles, err := readDirNames(helmlib.CatalogCache)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	iconCacheFiles, err := readDirNames(helmlib.IconCache)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(catalogCacheFiles) == 0 && len(iconCacheFiles) == 0 {
		return nil
	}

	var allCatalogs []*v3.Catalog
	var catalogHashes = map[string]bool{}

	catalogs, err := c.catalogClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	clusterCatalogs, err := c.clusterCatalogClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	projectCatalogs, err := c.projectCatalogClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, catalog := range catalogs.Items {
		allCatalogs = append(allCatalogs, &catalog)
	}
	for _, clusterCatalog := range clusterCatalogs.Items {
		allCatalogs = append(allCatalogs, &clusterCatalog.Catalog)
	}
	for _, projectCatalog := range projectCatalogs.Items {
		allCatalogs = append(allCatalogs, &projectCatalog.Catalog)
	}

	if len(targetCatalogs) == 0 {
		for _, catalog := range allCatalogs {
			catalogHashes[helmlib.CatalogSHA256Hash(catalog)] = true
		}
	}
	for _, catalog := range targetCatalogs {
		if contains(allCatalogs, catalog) {
			catalogHashes[helmlib.CatalogSHA256Hash(catalog)] = true
		}
	}

	var cleanupCount int
	cleanupCount += cleanupPath(helmlib.CatalogCache, catalogCacheFiles, catalogHashes)
	cleanupCount += cleanupPath(helmlib.IconCache, iconCacheFiles, catalogHashes)
	if cleanupCount > 0 {
		logrus.Infof("Catalog-cache removed %d entries from disk", cleanupCount)
	}
	return nil
}

func (c *CacheCleaner) Cleanup() error {
	logrus.Debug("Catalog-cache running cleanup")
	return c.cleanUpCatalogs(nil)
}

func (c *CacheCleaner) CleanTarget(catalog *v3.Catalog) error {
	logrus.Debug("Catalog-cache running cleanTarget:%v", catalog.Name)
	return c.cleanUpCatalogs([]*v3.Catalog{catalog})
}

func readDirNames(dir string) ([]string, error) {
	pathFile, err := os.Open(dir)
	defer pathFile.Close()
	if err != nil {
		return nil, err
	}
	return pathFile.Readdirnames(0)
}

func cleanupPath(dir string, files []string, valid map[string]bool) int {
	var count int
	for _, file := range files {
		if valid[file] || strings.HasPrefix(file, ".") {
			continue
		}

		dirFile := filepath.Join(dir, file)
		helmlib.Locker.Lock(file)
		os.RemoveAll(dirFile)
		helmlib.Locker.Unlock(file)

		count++
		logrus.Debugf("Catalog-cache removed entry from disk: %s", dirFile)
	}
	return count
}
