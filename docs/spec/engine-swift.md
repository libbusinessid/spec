# Prompt d’implémentation — moteur Swift LibBusinessID

## Mission

Implémente intégralement le dépôt Swift du moteur LibBusinessID. Lis d’abord
`engine.md` en entier. Avant de coder, exige et lis aussi `rules.proto`,
`conformance.proto`, `ir.md`, `features.md`, le manifeste, `SHA256SUMS`,
l’attestation et les deux BINPB de la même release. Si un élément manque ou se
contredit, arrête-toi et fais corriger `spec` ; ne déduis jamais une sémantique du
code Go de référence. `engine.md` définit la sémantique normative commune. Ce
document fixe les choix Swift. Le résultat doit être un Swift Package de production, idiomatique,
sûr sous concurrence stricte et agréable à utiliser dans une application iOS.

Ne traduis pas mécaniquement le moteur Go ou un autre runtime. Conçois les types,
erreurs, protocoles et tests selon les conventions Swift.

## Cibles et packaging

- `Package.swift` DOIT être à la racine du dépôt.
- Utiliser Swift Package Manager comme distribution principale.
- Swift 6 ou version minimale actuelle justifiée ; déclarer clairement les minimums
  iOS, macOS, tvOS et watchOS réellement testés.
- Tester au minimum macOS et un simulateur iOS.
- Le cœur doit éviter UIKit/AppKit et rester indépendant de l’UI.
- Utiliser `apple/swift-protobuf` pour le décodage.
- Le bundle et la conformité sont des ressources SPM accessibles via `Bundle.module`.
- Le module public s’appelle `BusinessID`.

Structure recommandée :

```text
.
├── Package.swift
├── Sources/BusinessID/
│   ├── API/
│   ├── Domain/
│   ├── Runtime/
│   ├── Rules/
│   ├── Registry/
│   ├── Generated/
│   └── Resources/businessid-rules.binpb
├── Tests/BusinessIDTests/
│   ├── Unit/
│   ├── Conformance/
│   ├── Security/
│   └── Resources/businessid-conformance.binpb
├── Examples/
└── rules.lock
```

## API Swift attendue

Concevoir une API proche de :

```swift
public struct IdentifierInput: Sendable, Hashable, Codable { ... }
public struct ValidationOptions: Sendable, Hashable { ... }
public struct CanonicalizationResult: Sendable, Hashable, Codable { ... }
public struct ValidationReport: Sendable, Hashable, Codable { ... }

public final class BusinessIDEngine: @unchecked Sendable {
    public static let `default`: BusinessIDEngine

    public func canonicalize(
        _ input: IdentifierInput,
        options: ValidationOptions = .init()
    ) -> CanonicalizationResult
    public func validate(
        _ input: IdentifierInput,
        options: ValidationOptions = .init()
    ) -> ValidationReport
    public func validateFormat(
        _ input: IdentifierInput,
        options: ValidationOptions = .init()
    ) -> ValidationReport
    public func validateChecksum(
        _ input: IdentifierInput,
        options: ValidationOptions = .init()
    ) -> ValidationReport
}
```

Il n'y a pas d'initialiseur acceptant un bundle en octets à l'exécution. Une version
antérieure de ce document déclarait `public init(rules: Data) throws`, ce qui imposait
de porter le validateur complet et la machine d'exécution de l'IR chez chaque
appelant — c'est-à-dire un interpréteur, que `engine.md` section 1.1 interdit. Un jeu
de règles personnalisé passe par le générateur, à la construction.

N’utilise `@unchecked Sendable` que si l’immuabilité est démontrée et documentée ;
préfère une `struct Sendable` ou un stockage interne réellement `Sendable` lorsque
possible. Aucun état mutable partagé après initialisation.

Principes API :

- value types (`struct`, `enum`) pour entrées et résultats ;
- enums exhaustifs et `Sendable` ;
- `IdentifierKind` est une `struct RawRepresentable` chaîne extensible avec valeurs
  statiques connues, pas un enum fermé ;
- `throws` pour construction/chargement invalide, pas pour une entrée métier
  invalide ;
- pas d’optional ambigu : `StepStatus.notRun` et `.unsupported` sont explicites ;
- types Protobuf strictement `internal` ;
- documentation DocC sur toute API publique ;
- propriétés calculées clairement nommées, sans `isValid` ambigu sur le rapport ;
- sérialisation Codable conforme aux chaînes de `engine.md` si exposée.

Le moteur par défaut doit charger la ressource une fois, de façon déterministe et
thread-safe. Une absence/corruption de la ressource doit être détectée par les tests
de packaging et ne doit pas devenir un crash déclenché par une entrée utilisateur.

## Interface registre Swift

Définir un protocole sans implémentation :

```swift
public protocol RegistryProvider: Sendable {
    func supports(kind: IdentifierKind, countryCode: String?) -> Bool
    func lookup(_ input: RegistryInput) async throws -> RegistryResult
}
```

- aucune dépendance URLSession dans le cœur ;
- aucun provider concret ;
- `async throws` réservé au registre ;
- distinguer `notFound` de l’erreur technique ;
- aucune invocation depuis `validate`.

## SwiftProtobuf

- Code généré sous `Sources/BusinessID/Generated`, visibilité internal.
- Version de `protoc-gen-swift` alignée avec le runtime et verrouillée.
- Commandes de génération et de vérification documentées.
- Ne pas exposer `SwiftProtobuf.Message` publiquement.
- Le bundle est lu comme `Data` puis validé entièrement avant construction du
  runtime immuable.
- Vérifier tailles et conversions `Int`, `UInt32`, indices de collections.
- Ne jamais supposer que les unknown fields rendent un bundle compatible.

## Style Swift et sûreté

- Swift 6 strict concurrency activée ;
- aucune data race ou global mutable ;
- aucun `try!`, `as!` ou force unwrap dans le cœur ;
- une force unwrap éventuelle dans un test doit être locale et justifiée ;
- pas de dépendance à `Locale.current` ;
- uppercase et whitespace conformes explicitement à `engine.md` ;
- ne pas indexer `String` par offsets UTF-16 ;
- implémenter les vues ASCII de façon sûre après validation ;
- erreurs structurées conformes à `Error`, `Sendable`, `Equatable` si possible ;
- méthodes courtes, noms selon Swift API Design Guidelines ;
- préférer immutabilité et composition ;
- éviter Objective-C bridging dans le cœur ;
- aucune regex pour interpréter les règles.

Configurer SwiftFormat et SwiftLint, ou outils équivalents maintenus. Le choix et les
versions sont verrouillés. Zéro warning de compilation et de lint. Toute suppression
est locale et commentée.

## Tests Swift obligatoires

Utiliser Swift Testing ou XCTest selon la version minimale, avec une convention
unique. Les tests couvrent :

- chaque opération IR et chaque branche ;
- décodage/validation de bundles malveillants ;
- ressource SPM dans un vrai test package ;
- API publique et Codable ;
- conformité commune complète ;
- exécution parallèle avec task groups ;
- strict concurrency ;
- canonicalisation Unicode/ASCII exacte ;
- overflow et bornes d’indices ;
- interface registre mockée, sans réseau ;
- exemples DocC compilables.

Ajouter property-based testing avec une dépendance maintenue ou générateurs internes
simples. Ajouter un harness de fuzzing Swift pour le décodeur et les opérations
critiques, exécutable en CI planifiée. Les tests de conformité ne doivent pas être
réécrits manuellement en Swift : ils sont lus depuis le BINPB.

Quality gates :

- couverture lignes ≥ 95 % via `swift test --enable-code-coverage` ;
- couverture branches maximale selon outils disponibles ;
- 100 % dispatch IR et refus version/feature ;
- tests sur version Swift minimale et actuelle ;
- aucune exclusion du cœur ;
- mutation testing si un outil stable compatible est disponible, sinon tests de
  mutation manuels ciblés sur checksums et bornes.

## Performance

- bundle décodé une fois ;
- API locale synchrone ;
- limiter copies de `String`/`Data` sans introduire `unsafe` ;
- benchmarks XCTest/Swift Benchmark pour cold load, validations et parallélisme ;
- aucune optimisation non prouvée par profil et tests ;
- éviter de conserver le graph Protobuf brut si une représentation interne plus sûre
  et compacte est préférable.

## CI et publication

Créer GitHub Actions pour :

- `swift build` debug/release ;
- tests macOS ;
- tests simulateur iOS avec Xcode supporté ;
- version Swift minimale et actuelle lorsque possible ;
- SwiftFormat/SwiftLint ;
- stricte concurrence et warnings-as-errors ;
- couverture et seuil ;
- conformité ;
- fuzz smoke et audit dépendances ;
- vérification code Protobuf généré ;
- test d’intégration d’une application/package consommateur ;
- validation que `Package.swift` et les tags SemVer permettent la résolution SPM.

La release produit un tag SemVer complet. Le README précise les plateformes et
versions réellement testées.

## Documentation

Fournir README et DocC avec : installation SPM, exemple de validation, format seul,
checksum unsupported, moteur personnalisé, versions, concurrence, registre futur et
limite « aucune preuve d’existence ». Ajouter SECURITY, CONTRIBUTING, changelog et
un exemple iOS minimal sans en faire une dépendance du package.

## Interdictions spécifiques

- pas de singleton mutable ;
- pas de dépendance UIKit/AppKit dans le cœur ;
- pas de `fatalError` atteignable depuis une entrée non fiable ;
- pas de force unwrap pour les indices/données Protobuf ;
- pas d’API publique en types générés ;
- pas de `NSRegularExpression` pour les règles ;
- pas de copie naïve d’interfaces Go/Kotlin ;
- pas de provider registre concret ;
- pas de réduction des cas ou seuils.

## Livrables et définition de terminé

Livrer package SPM complet, API, moteur, ressources, Protobuf généré, interface
registre, tests, fuzz harness, benchmarks, CI, DocC et documentation de release.
Tous les critères de `engine.md` doivent être satisfaits ; build release, lint,
conformité, couverture, tests de concurrence et test consommateur doivent être verts,
sans TODO V1 ni warning.
