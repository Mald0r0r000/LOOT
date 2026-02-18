class Loot < Formula
  desc "Professional Media Offload Tool (TUI) for secure data transfer"
  homepage "https://github.com/Mald0r0r000/LOOT"
  license "MIT"

  # Stable Release via Git Tag
  url "https://github.com/Mald0r0r000/LOOT.git",
      tag:      "v1.0.8",
      revision: "e1becc27f75bf751150e44d640ef39c04f6d9f0d"
  
  head "https://github.com/Mald0r0r000/LOOT.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build the binary
    # std_go_args automatically sets install path to #{bin}/loot
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/loot"
  end

  test do
    # Simple verification test
    assert_match version.to_s, shell_output("#{bin}/loot --version")
  end
end
