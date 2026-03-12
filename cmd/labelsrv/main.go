package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ostretsov/labelsrv/internal/api"
	"github.com/ostretsov/labelsrv/internal/renderer"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
	"github.com/ostretsov/labelsrv/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "labelsrv",
	Short: "Configuration-driven label rendering server",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP label rendering server",
		RunE:  runServe,
	}
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serveCmd.Flags().String("labels", "labels", "Directory containing label templates")
	serveCmd.Flags().String("fonts", "", "Directory with extra TTF fonts (optional)")

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the server with hot-reload on template changes",
		RunE:  runDev,
	}
	devCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	devCmd.Flags().String("labels", "labels", "Directory containing label templates")
	devCmd.Flags().String("fonts", "", "Directory with extra TTF fonts (optional)")

	renderCmd := &cobra.Command{
		Use:   "render <template.yaml> <data.json>",
		Short: "Render a label template with provided data to PDF",
		Args:  cobra.ExactArgs(2),
		RunE:  runRender,
	}
	renderCmd.Flags().StringP("output", "o", "label.pdf", "Output PDF file path")
	renderCmd.Flags().String("fonts", "", "Directory with extra TTF fonts (optional)")

	validateCmd := &cobra.Command{
		Use:   "validate <template.yaml>",
		Short: "Validate a label template file",
		Args:  cobra.ExactArgs(1),
		RunE:  runValidate,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Version)
		},
	}

	rootCmd.AddCommand(serveCmd, devCmd, renderCmd, validateCmd, versionCmd)
}

func startServer(port int, labelsDir, fontsDir string, dev bool) error {
	loader := tmpl.NewTemplateLoader()
	if err := loader.LoadAll(labelsDir); err != nil {
		return fmt.Errorf("loading templates from %q: %w", labelsDir, err)
	}

	if dev {
		loader.Watch(labelsDir)
	}

	r, err := renderer.New(fontsDir)
	if err != nil {
		return fmt.Errorf("initialising renderer: %w", err)
	}

	mux := api.New(loader, r)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(ctx)
	}()

	prefix := ""
	if dev {
		prefix = "[dev] "
	}

	fmt.Printf("%slabelsrv %s\n", prefix, version.Version)
	fmt.Printf("  listening on  :%d\n", port)
	fmt.Printf("  labels dir    %s\n", labelsDir)

	if fontsDir != "" {
		fmt.Printf("  fonts dir     %s\n", fontsDir)
	}

	if dev {
		fmt.Printf("  watching      %s\n", labelsDir)
	}

	templates := loader.List()
	fmt.Printf("  templates (%d)\n", len(templates))

	for _, name := range templates {
		fmt.Printf("    - %s\n", name)
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: %w", err)
	}

	return nil
}

func runServe(cmd *cobra.Command, _ []string) error {
	port, _ := cmd.Flags().GetInt("port")
	labelsDir, _ := cmd.Flags().GetString("labels")
	fontsDir, _ := cmd.Flags().GetString("fonts")

	return startServer(port, labelsDir, fontsDir, false)
}

func runDev(cmd *cobra.Command, _ []string) error {
	port, _ := cmd.Flags().GetInt("port")
	labelsDir, _ := cmd.Flags().GetString("labels")
	fontsDir, _ := cmd.Flags().GetString("fonts")

	return startServer(port, labelsDir, fontsDir, true)
}

func runRender(cmd *cobra.Command, args []string) error {
	templatePath := args[0]
	dataPath := args[1]
	outputPath, _ := cmd.Flags().GetString("output")
	fontsDir, _ := cmd.Flags().GetString("fonts")

	t, err := tmpl.ParseFile(templatePath)
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	if err := tmpl.Validate(t); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	dataBytes, err := os.ReadFile(dataPath) //nolint:gosec // CLI arg is trusted input
	if err != nil {
		return fmt.Errorf("reading data file: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("parsing data JSON: %w", err)
	}

	r, err := renderer.New(fontsDir)
	if err != nil {
		return fmt.Errorf("initialising renderer: %w", err)
	}

	pdfBytes, err := r.Render(t, data)
	if err != nil {
		return fmt.Errorf("rendering label: %w", err)
	}

	if err := os.WriteFile(outputPath, pdfBytes, 0600); err != nil {
		return fmt.Errorf("writing output PDF: %w", err)
	}

	fmt.Printf("rendered label to %s (%d bytes)\n", outputPath, len(pdfBytes))

	return nil
}

func runValidate(_ *cobra.Command, args []string) error {
	t, err := tmpl.ParseFile(args[0])
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	if err := tmpl.Validate(t); err != nil {
		fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("template %q is valid\n", t.Name)

	return nil
}
