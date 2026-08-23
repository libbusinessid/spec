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
- Utiliser `apple/swift-protobuf` pour le décodage **dans la cible générateur**.
  SwiftPM résout tout le manifeste : un consommateur récupère la dépendance même
  s'il ne la lie pas, donc annoncer « aucune dépendance » sans le vérifier est faux.
- Le bundle n'est pas une ressource du paquet : il est lu par le générateur à la
  construction. Le corpus de conformité reste accessible aux tests, jamais à la
  bibliothèque.
- Le module public s’appelle `BusinessID`.

Structure recommandée :

```text
.
├── Package.swift
├── Sources/BusinessID/
│   ├── API/
│   ├── Domain/
│   ├── Runtime/          # primitives que le code généré appelle
│   └── Generated/        # code émis depuis le bundle, jamais édité
├── Sources/businessid-gen/   # le générateur : lit le bundle, valide, émet
├── Tests/BusinessIDTests/
│   ├── Unit/
│   ├── Conformance/
│   └── Security/
├── Examples/
└── rules.lock
```

Le générateur et la bibliothèque sont deux cibles. Le décodage Protobuf, les
vingt-cinq contrôles de chargement et la connaissance des opcodes vivent dans le
générateur ; la bibliothèque publiée ne contient que le code émis, les primitives
qu'il appelle et l'API.

Aucun `.binpb` n'est une ressource du paquet. Le bundle est une entrée du générateur,
lue à la construction — l'embarquer comme `Resources` ferait porter à chaque appelant
une donnée que le code généré rend inutile, et supposerait un décodeur pour la lire.

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
appelant — c'est-à-dire un interpréteur, que `engine.md` section 1.2 interdit. Un jeu
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

Le moteur par défaut ne charge rien : il est le code généré, disponible sans
initialisation coûteuse ni verrou, et utilisable depuis plusieurs tâches. Aucune
entrée utilisateur ne peut le faire échouer, puisque tout ce qu'il contient a été
produit à partir d'un bundle déjà accepté.

### Registre : rien à livrer en V1

Ce moteur n'expose aucun type de registre. `engine.md` section 10 diffère la
consultation d'un registre distant à une version ultérieure, et un moteur qui n'en
porte pas est pleinement conforme.

Trois choses à ne pas faire, parce qu'elles fermeraient la porte :

- n'exposez aucun type de registre, même marqué expérimental : un type public est un
  engagement que SemVer fige ;
- ne rendez aucune méthode de validation asynchrone « au cas où » — la validation
  locale reste synchrone définitivement ;
- ne mettez aucune dépendance HTTP dans le paquet du cœur.

Une consultation de registre porte un jeton d'API : elle ne devra jamais être possible
depuis un navigateur, et vivra donc dans un module séparé, chargé côté serveur
uniquement.


## SwiftProtobuf

- Code généré sous `Sources/BusinessID/Generated`, visibilité internal, jamais édité à la main.
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
- décodage/validation de bundles malveillants, dans le générateur ;
- packaging : que le paquet publié ne porte ni `.binpb` ni SwiftProtobuf ;
- API publique et Codable ;
- conformité commune complète ;

Le runner de conformité vient de `spec`, jamais de ce dépôt. Il s'exécute épinglé au
commit que `rules.lock` enregistre sous `source_commit` :

```bash
go run github.com/libbusinessid/spec/cmd/conformance-runner@<source_commit> \
  -corpus spec/businessid-conformance.binpb -- .build/debug/businessid-testee
```

N'écris pas de comparateur : un moteur qui juge lui-même ses propres résultats peut
se déclarer conforme en comparant trop faiblement. Ce que tu écris, c'est le testee,
et les tests qui prouvent qu'il ne triche pas.

`invalid_encoding` ne peut pas être un cas de conformité — `ir.md` section 5 étape 1
dit pourquoi. Épingle-le par un test natif : `String` étant toujours bien formé en Swift,
la branche est inatteignable par cette API. Documente-le, avec la raison, plutôt
que d'ajouter un point d'entrée orienté octets qui n'existerait que pour elle.

- exécution parallèle avec task groups ;
- strict concurrency ;
- canonicalisation Unicode/ASCII exacte ;
- overflow et bornes d’indices ;
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
- vérification du code émis par le générateur, régénération byte-identique ;
- test d’intégration d’une application/package consommateur ;
- validation que `Package.swift` et les tags SemVer permettent la résolution SPM.

La release produit un tag SemVer complet. Le README précise les plateformes et
versions réellement testées.

## Documentation

Fournir README et DocC avec : installation SPM, exemple de validation, format seul,
checksum unsupported, jeu de règles personnalisé passant par le générateur,
versions, concurrence et
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
- pas de réduction des cas ou seuils.

## Livrables et définition de terminé

Livrer package SPM complet, générateur, API, moteur, code émis, tests, fuzz harness,
benchmarks, CI, DocC et documentation de release. Aucune ressource `.binpb` dans le
paquet, aucun type de registre.
Tous les critères de `engine.md` doivent être satisfaits ; build release, lint,
conformité, couverture, tests de concurrence et test consommateur doivent être verts,
sans TODO V1 ni warning.
