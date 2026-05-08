binary {
  secrets {
    all = true
  }
  go_modules   = true
  osv          = true
  oss_index    = false
  nvd          = false
  
  triage {
    suppress {
      vulnerabilities = [
        "GO", 
        "GHSA"
      ]
    }
  }
}
