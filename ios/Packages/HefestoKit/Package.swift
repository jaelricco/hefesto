// swift-tools-version:6.1
//
// Everything in the iOS app that is not a view. Each target builds and tests
// with `swift test` on macOS and Linux; the app target in ../../Hefesto holds
// only SwiftUI views. See docs/adr/0010-ios-architecture.md.

import PackageDescription

let package = Package(
    name: "HefestoKit",
    defaultLocalization: "en",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "HefestoAPI", targets: ["HefestoAPI"]),
        .library(name: "HefestoStore", targets: ["HefestoStore"]),
        .library(name: "HefestoAuth", targets: ["HefestoAuth"]),
        .library(name: "HefestoSync", targets: ["HefestoSync"]),
        .library(name: "HefestoLogger", targets: ["HefestoLogger"]),
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-openapi-generator", from: "1.13.0"),
        .package(url: "https://github.com/apple/swift-openapi-runtime", from: "1.12.0"),
        .package(url: "https://github.com/apple/swift-openapi-urlsession", from: "1.3.0"),
        .package(url: "https://github.com/groue/GRDB.swift", from: "7.11.0"),
    ],
    targets: [
        // Generated from api/openapi.yaml (symlinked). DTOs are never written by hand.
        .target(
            name: "HefestoAPI",
            dependencies: [
                .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
                .product(name: "OpenAPIURLSession", package: "swift-openapi-urlsession"),
            ],
            plugins: [.plugin(name: "OpenAPIGenerator", package: "swift-openapi-generator")]
        ),
        .target(
            name: "HefestoStore",
            dependencies: [.product(name: "GRDB", package: "GRDB.swift")]
        ),
        .target(
            name: "HefestoAuth",
            dependencies: ["HefestoAPI"]
        ),
        .target(
            name: "HefestoSync",
            dependencies: ["HefestoAPI", "HefestoStore"]
        ),
        .target(
            name: "HefestoLogger",
            dependencies: ["HefestoStore"]
        ),
        .testTarget(name: "HefestoStoreTests", dependencies: ["HefestoStore"]),
        .testTarget(name: "HefestoAuthTests", dependencies: ["HefestoAuth", "HefestoAPI"]),
        .testTarget(name: "HefestoSyncTests", dependencies: ["HefestoSync", "HefestoStore", "HefestoAPI"]),
        .testTarget(name: "HefestoLoggerTests", dependencies: ["HefestoLogger", "HefestoStore"]),
    ]
)
