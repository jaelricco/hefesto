import HefestoStore
import HefestoSync
import SwiftUI

// Choosing the band a set was assisted with (ADR 0024). Bands are created
// online (ADR 0009); the catalogue is empty until real manufacturer data is
// researched, so most bands are the athlete's own.

/// The bands to choose from: the athlete's own first, then the catalogue's.
struct BandPicker: View {
    let db: AppDatabase
    /// The band chosen so far, if any.
    let selected: String?
    let onPick: (Band) -> Void

    @Environment(AppModel.self) private var app
    @Environment(\.dismiss) private var dismiss
    @State private var bands: [Band] = []
    @State private var adding = false
    @State private var failure: String?

    var body: some View {
        NavigationStack {
            List {
                if bands.isEmpty {
                    Text("No bands yet. Add the ones you train with.")
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)
                        .themedRow()
                }
                let mine = bands.filter(\.isMine)
                let catalogue = bands.filter { !$0.isMine }
                if !mine.isEmpty {
                    Section {
                        ForEach(mine) { band in
                            row(band)
                                .swipeActions {
                                    Button("Remove", role: .destructive) { Task { await remove(band) } }
                                }
                        }
                    } header: {
                        CapsLabel("My bands")
                    }
                }
                if !catalogue.isEmpty {
                    Section {
                        ForEach(catalogue) { row($0) }
                    } header: {
                        CapsLabel("Catalogue")
                    }
                }
                Section {
                    Button {
                        adding = true
                    } label: {
                        Label("Add a band", systemImage: "plus")
                    }
                    .font(.bodyMedium)
                    .foregroundStyle(Palette.ember)
                    .frame(minHeight: 44)
                    .themedRow()
                } footer: {
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                }
            }
            .themedScreen()
            .navigationTitle("Band")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
            .navigationDestination(isPresented: $adding) {
                AddBandForm { band in
                    onPick(band)
                    dismiss()
                }
            }
            .task {
                do { for try await b in db.observeBands() { bands = b } } catch {}
            }
        }
    }

    private func row(_ band: Band) -> some View {
        Button {
            onPick(band)
            dismiss()
        } label: {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(verbatim: BandText.name(band)).font(.rowTitle).foregroundStyle(Palette.text)
                    Text(verbatim: BandText.resistance(band)).font(.fine).foregroundStyle(Palette.textSecondary)
                }
                Spacer()
                if band.id == selected {
                    Image(systemName: "checkmark").foregroundStyle(Palette.text)
                }
            }
            .frame(minHeight: 44)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .themedRow()
        .accessibilityAddTraits(band.id == selected ? .isSelected : [])
    }

    private func remove(_ band: Band) async {
        failure = nil
        do {
            try await app.deleteBand(id: band.id)
        } catch {
            failure = BandText.failure(error)
        }
    }
}

/// A new band of the athlete's own: brand, colour and what the maker states.
struct AddBandForm: View {
    let onAdded: (Band) -> Void

    @Environment(AppModel.self) private var app
    @State private var brand = ""
    @State private var colour = ""
    @State private var minKg: Double?
    @State private var maxKg: Double?
    @State private var saving = false
    @State private var failure: String?

    var body: some View {
        Form {
            Section {
                TextField("Brand", text: $brand)
                TextField("Colour or name", text: $colour)
            }
            .themedRow()
            Section {
                LabeledContent("From") {
                    TextField("kg", value: $minKg, format: .number)
                        .keyboardType(.decimalPad)
                        .multilineTextAlignment(.trailing)
                }
                LabeledContent("To") {
                    TextField("kg", value: $maxKg, format: .number)
                        .keyboardType(.decimalPad)
                        .multilineTextAlignment(.trailing)
                }
            } header: {
                CapsLabel("Resistance as the maker states it")
            } footer: {
                Text("Makers often state more than a band gives, and bands of one colour differ.")
                    .font(.detailText)
            }
            .themedRow()
            if let failure {
                Section {
                    Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                }
                .themedRow()
            }
        }
        .themedScreen()
        .font(.bodyText)
        .navigationTitle("New band")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("Add") { Task { await save() } }
                    .fontWeight(.semibold)
                    .disabled(!valid || saving)
            }
        }
    }

    private var valid: Bool {
        guard let minKg, let maxKg else { return false }
        return !trimmed(brand).isEmpty && !trimmed(colour).isEmpty && minKg >= 0 && minKg <= maxKg
    }

    private func trimmed(_ s: String) -> String { s.trimmingCharacters(in: .whitespacesAndNewlines) }

    private func save() async {
        guard let minKg, let maxKg else { return }
        saving = true
        failure = nil
        do {
            let band = try await app.createBand(
                brand: trimmed(brand), colourLabel: trimmed(colour), minKg: minKg, maxKg: maxKg)
            onAdded(band)
        } catch {
            failure = BandText.failure(error)
        }
        saving = false
    }
}

/// A band as the athlete reads it.
enum BandText {
    /// "Brand A Red".
    static func name(_ band: Band) -> String {
        "\(band.brand) \(band.colourLabel)"
    }

    /// "5–15 kg", as the maker states it.
    static func resistance(_ band: Band) -> String {
        let format = FloatingPointFormatStyle<Double>.number.precision(.fractionLength(0...1))
        return String(localized: "\(band.resistanceMinKg.formatted(format))–\(band.resistanceMaxKg.formatted(format)) kg")
    }

    /// The band of an element, if it names one: "Band: Brand A Red ×2".
    static func assistance(_ a: Assistance, bands: [String: Band]) -> String? {
        guard a.type == "band", let id = a.bandId, let band = bands[id] else { return nil }
        return a.bandCount > 1
            ? String(localized: "Band: \(name(band)) ×\(a.bandCount)")
            : String(localized: "Band: \(name(band))")
    }

    static func failure(_ error: any Error) -> String {
        if let e = error as? BandError {
            return switch e {
            case .duplicate: String(localized: "You have a band of this brand and colour already.")
            case .invalid: String(localized: "Check the figures: the first must not be above the second.")
            case .refused: String(localized: "The band could not be saved. Try again later.")
            }
        }
        return String(localized: "Bands are saved online. Try again when you have a connection.")
    }
}
