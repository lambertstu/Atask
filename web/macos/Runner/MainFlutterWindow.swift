import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    
    // Set initial window size
    self.setContentSize(NSSize(width: 1400, height: 1000))
    
    self.contentViewController = flutterViewController
    
    // Set minimum window size to prevent layout overflow
    self.minSize = NSSize(width: 1200, height: 800)
    
    // Center the window on screen
    self.center()

    RegisterGeneratedPlugins(registry: flutterViewController)

    super.awakeFromNib()
  }
}