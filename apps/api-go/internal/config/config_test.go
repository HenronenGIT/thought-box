package config

import "testing"

func TestFromMapRequiresPostgresURLWithCredentials(t *testing.T) {
	base := []string{
		"OPENAI_API_KEY=test",
		"S3_BUCKET=thoughts",
		"S3_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	}

	_, err := fromMap(append(base, "DATABASE_URL=jdbc:postgresql://localhost:5432/thoughts_dev"))
	if err == nil {
		t.Fatal("expected invalid URL")
	}

	cfg, err := fromMap(append(base, "DATABASE_URL=postgres://thoughts:thoughts@localhost:5432/thoughts_dev"))
	if err != nil {
		t.Fatalf("expected config, got %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port, got %s", cfg.Port)
	}
}
