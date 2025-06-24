config {
  plugin_dir = "./.tflint.d/plugins"  # configブロック内
}

plugin "google" {
  enabled = true
  version = "0.33.0"
  source  = "github.com/terraform-linters/tflint-ruleset-google"
}