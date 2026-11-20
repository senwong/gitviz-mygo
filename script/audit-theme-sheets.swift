import AppKit
struct Capture: Decodable {let Theme:String;let Scene:String;let Size:String;let File:String}
let root=URL(fileURLWithPath:CommandLine.arguments[1])
let captures=try JSONDecoder().decode([Capture].self,from:Data(contentsOf:root.appendingPathComponent("manifest.json")))
let out=root.appendingPathComponent("sheets");try FileManager.default.createDirectory(at:out,withIntermediateDirectories:true)
func sheet(_ items:[Capture],_ name:String,_ cols:Int=4) throws {
 let cw=460,ch=324,rows=(items.count+cols-1)/cols,w=cw*cols,h=ch*rows
 let rep=NSBitmapImageRep(bitmapDataPlanes:nil,pixelsWide:w,pixelsHigh:h,bitsPerSample:8,samplesPerPixel:4,hasAlpha:true,isPlanar:false,colorSpaceName:.deviceRGB,bytesPerRow:0,bitsPerPixel:0)!
 NSGraphicsContext.saveGraphicsState();NSGraphicsContext.current=NSGraphicsContext(bitmapImageRep:rep)
 NSColor(calibratedWhite:0.18,alpha:1).setFill();NSRect(x:0,y:0,width:w,height:h).fill()
 for (i,c) in items.enumerated(){let x=i%cols*cw,y=h-(i/cols+1)*ch
  let title="\(c.Theme) · \(c.Scene) · \(c.Size)"
  (title as NSString).draw(in:NSRect(x:x+6,y:y+ch-21,width:cw-12,height:18),withAttributes:[.font:NSFont.systemFont(ofSize:12),.foregroundColor:NSColor.white])
  if let img=NSImage(contentsOf:root.appendingPathComponent(c.File)){let scale=min(CGFloat(cw)/img.size.width,CGFloat(ch-26)/img.size.height);img.draw(in:NSRect(x:CGFloat(x),y:CGFloat(y),width:img.size.width*scale,height:img.size.height*scale))}
 }
 NSGraphicsContext.restoreGraphicsState()
 try rep.representation(using:.png,properties:[:])!.write(to:out.appendingPathComponent(name+".png"))
}
let themes=Array(Set(captures.map{$0.Theme})).sorted()
let scenes=Array(Set(captures.map{$0.Scene})).sorted()
for theme in themes {for size in ["wide","minimum"]{let items=captures.filter{$0.Theme==theme && $0.Size==size}.sorted{$0.Scene<$1.Scene};for start in stride(from:0,to:items.count,by:16){try sheet(Array(items[start..<min(start+16,items.count)]),"theme-\(theme)-\(size)-\(start/16)")}}}
for scene in scenes {try sheet(captures.filter{$0.Scene==scene && $0.Size=="wide"}.sorted{$0.Theme<$1.Theme},"scene-\(scene)")}
print("Generated sheets for \(captures.count) screenshots")
