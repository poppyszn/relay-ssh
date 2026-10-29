// Renders Relay's icon artwork from its geometry (macOS only: uses CoreGraphics and AppKit).
//
//   swiftc -O -o /tmp/make-icon desktop/build/icon/make-icon.swift
//   /tmp/make-icon tile    desktop/build/appicon.png          # 1024 macOS-style tile with margin
//   /tmp/make-icon full    /tmp/relay-full.png                # 2000 full-bleed square, for comparison
//   /tmp/make-icon social  docs/assets/social-preview.png     # 1280x640 GitHub social preview
//
// The symbol lives on a 2000-unit grid: a blue block with a chevron (a terminal prompt) cut out of
// it as negative space, and a lighter block behind it, like a signal handed on.
import AppKit
import CoreGraphics

// Colours are explicit sRGB: CGColor(red:green:blue:) is not sRGB and shifts the result.
let srgb = CGColorSpace(name: CGColorSpace.sRGB)!
func hex(_ r: Int, _ g: Int, _ b: Int) -> CGColor {
    CGColor(colorSpace: srgb, components: [CGFloat(r) / 255, CGFloat(g) / 255, CGFloat(b) / 255, 1])!
}
let navy = hex(0x0F, 0x1A, 0x28)
let blue = hex(0x31, 0x8E, 0xDC)
let light = hex(0x7A, 0xB2, 0xEE)

let block: [(Double, Double)] = [(726, 500), (1250, 500), (1250, 1500), (750, 1500), (750, 1250), (500, 1250), (500, 726)]
let chevron: [(Double, Double)] = [(750, 750), (904, 750), (1156, 1001), (904, 1250), (750, 1250), (1001, 1001)]
let behind: [(Double, Double)] = [(1250, 750), (1500, 750), (1500, 1250), (1250, 1500)]

func poly(_ pts: [(Double, Double)], _ ox: CGFloat, _ oy: CGFloat, _ s: CGFloat) -> CGPath {
    let p = CGMutablePath()
    for (i, pt) in pts.enumerated() {
        let q = CGPoint(x: ox + CGFloat(pt.0) * s, y: oy + CGFloat(pt.1) * s)
        i == 0 ? p.move(to: q) : p.addLine(to: q)
    }
    p.closeSubpath()
    return p
}

/// Draws the symbol with its 2000-unit grid placed at (ox, oy) and scaled by s.
func symbol(_ ctx: CGContext, _ ox: CGFloat, _ oy: CGFloat, _ s: CGFloat) {
    ctx.setFillColor(blue)
    ctx.addPath(poly(block, ox, oy, s))
    ctx.addPath(poly(chevron, ox, oy, s))
    ctx.fillPath(using: .evenOdd)
    ctx.setFillColor(light)
    ctx.addPath(poly(behind, ox, oy, s))
    ctx.fillPath()
}

/// An Apple-like continuous-corner square (a superellipse).
func squircle(_ r: CGRect, n: Double = 5) -> CGPath {
    let p = CGMutablePath()
    let steps = 360
    for i in 0...steps {
        let t = Double(i) / Double(steps) * 2 * Double.pi
        let c = cos(t), s = sin(t)
        let x = r.midX + r.width / 2 * CGFloat(c.sign == .minus ? -1 : 1) * CGFloat(pow(abs(c), 2 / n))
        let y = r.midY + r.height / 2 * CGFloat(s.sign == .minus ? -1 : 1) * CGFloat(pow(abs(s), 2 / n))
        i == 0 ? p.move(to: CGPoint(x: x, y: y)) : p.addLine(to: CGPoint(x: x, y: y))
    }
    p.closeSubpath()
    return p
}

func makeContext(_ w: Int, _ h: Int) -> CGContext {
    let ctx = CGContext(data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: 0,
                        space: CGColorSpace(name: CGColorSpace.sRGB)!, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)!
    ctx.translateBy(x: 0, y: CGFloat(h)) // draw with y growing downward, like the grid
    ctx.scaleBy(x: 1, y: -1)
    ctx.setShouldAntialias(true)
    return ctx
}

func save(_ ctx: CGContext, _ path: String) {
    let rep = NSBitmapImageRep(cgImage: ctx.makeImage()!)
    try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: path))
}

/// The macOS-style tile: a 824pt squircle centred on a 1024pt transparent canvas, with a soft shadow.
func drawTile(_ ctx: CGContext, size: CGFloat, originX: CGFloat = 0, originY: CGFloat = 0) {
    let body = size * 824 / 1024
    let rect = CGRect(x: originX + (size - body) / 2, y: originY + (size - body) / 2, width: body, height: body)
    ctx.saveGState()
    ctx.setShadow(offset: CGSize(width: 0, height: -size * 0.012), blur: size * 0.03, color: CGColor(gray: 0, alpha: 0.35))
    ctx.setFillColor(navy)
    ctx.addPath(squircle(rect))
    ctx.fillPath()
    ctx.restoreGState()
    ctx.saveGState()
    ctx.addPath(squircle(rect))
    ctx.clip()
    ctx.setFillColor(navy)
    ctx.fill(rect)
    symbol(ctx, rect.minX, rect.minY, body / 2000)
    ctx.restoreGState()
}

let mode = CommandLine.arguments[1]
let out = CommandLine.arguments[2]
switch mode {
case "tile":
    let ctx = makeContext(1024, 1024)
    drawTile(ctx, size: 1024)
    save(ctx, out)
case "full":
    let ctx = makeContext(2000, 2000)
    ctx.setFillColor(navy)
    ctx.fill(CGRect(x: 0, y: 0, width: 2000, height: 2000))
    symbol(ctx, 0, 0, 1)
    save(ctx, out)
case "social":
    let w = 1280, h = 640
    let ctx = makeContext(w, h)
    ctx.setFillColor(navy)
    ctx.fill(CGRect(x: 0, y: 0, width: w, height: h))
    drawTile(ctx, size: 440, originX: 90, originY: 100)
    let gfx = NSGraphicsContext(cgContext: ctx, flipped: true)
    NSGraphicsContext.current = gfx
    func text(_ s: String, _ x: CGFloat, _ y: CGFloat, _ size: CGFloat, _ weight: NSFont.Weight, _ color: NSColor) {
        (s as NSString).draw(at: NSPoint(x: x, y: y), withAttributes: [.font: NSFont.systemFont(ofSize: size, weight: weight), .foregroundColor: color])
    }
    text("Relay", 600, 205, 150, .semibold, .white)
    text("SSH connection manager and terminal", 604, 372, 40, .regular, NSColor(srgbRed: 0.553, green: 0.608, blue: 0.690, alpha: 1))
    text("Built on OpenSSH  ·  Open source", 604, 430, 30, .regular, NSColor(srgbRed: 0.231, green: 0.604, blue: 0.882, alpha: 1))
    NSGraphicsContext.current = nil
    save(ctx, out)
default:
    fatalError("mode must be tile, full or social")
}
