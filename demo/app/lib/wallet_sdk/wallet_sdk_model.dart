class SubmissionRequirement {
  final String rule;
  final String name;
  final int min;
  final int max;
  final int count;
  final List<SubmissionRequirement> nested;
  final List<InputDescriptor> inputDescriptors;

//<editor-fold desc="Data Methods">
  const SubmissionRequirement({
    required this.rule,
    required this.name,
    required this.min,
    required this.max,
    required this.count,
    required this.nested,
    required this.inputDescriptors,
  });

  @override
  String toString() {
    return 'SubmissionRequirement{ rule: $rule, name: $name, min: $min, max: $max, count: $count, nested: $nested, inputDescriptors: $inputDescriptors,}';
  }

  factory SubmissionRequirement.fromMap(Map<String, dynamic> map) {
    return SubmissionRequirement(
      rule: map['rule'] as String,
      name: map['name'] as String,
      min: map['min'] as int,
      max: map['max'] as int,
      count: map['count'] as int,
      nested: (map['nested'] as List<dynamic>)
          .map((obj) => SubmissionRequirement.fromMap(obj.cast<String, dynamic>()))
          .toList(),
      inputDescriptors: (map['inputDescriptors'] as List<dynamic>)
          .map((obj) => InputDescriptor.fromMap(obj.cast<String, dynamic>()))
          .toList(),
    );
  }

//</editor-fold>
}

class InputDescriptor {
  final String id;
  final String name;
  final String purpose;
  final List<String> matchedVCsID;
  final List<String> matchedVCs;

  const InputDescriptor({
    required this.id,
    required this.name,
    required this.purpose,
    required this.matchedVCsID,
    required this.matchedVCs,
  });

  @override
  String toString() {
    return 'InputDescriptor{ id: $id, name: $name, purpose: $purpose, matchedVCs: $matchedVCs, matchedVCsID: $matchedVCsID }';
  }

  factory InputDescriptor.fromMap(Map<String, dynamic> map) {
    return InputDescriptor(
      id: map['id'] as String,
      name: map['name'] as String,
      purpose: map['purpose'] as String,
      matchedVCsID: map['matchedVCsID'].cast<String>(),
      matchedVCs: map['matchedVCs'].cast<String>(),
    );
  }
}

class SupportedCredentials {
  final String format;
  final List<String> types;

  const SupportedCredentials({
    required this.format,
    required this.types,
  });

  @override
  String toString() {
    return 'SupportedCredentials{ format: $format, types: $types }';
  }

  factory SupportedCredentials.fromMap(Map<String, dynamic> map) {
    return SupportedCredentials(
      format: map['format'] as String,
      types: map['types'].cast<String>(),
    );
  }
}

class WalletSDKError {
  final String code;
  final String category;
  final String details;
  final String traceID;

  const WalletSDKError({
    required this.code,
    required this.category,
    required this.details,
    required this.traceID,
  });

  @override
  String toString() {
    return 'WalletSDKError { code: $code, category: $category, details: $details, traceID: $traceID }';
  }

  Map<String, dynamic> toJson() {
    final Map<String, dynamic> data = <String, dynamic>{};
    data['code'] = code;
    data['category'] = category;
    data['details'] = details;
    data['traceID'] = traceID;
    return data;
  }

  factory WalletSDKError.fromJson(Map<String, dynamic> json) {
    return WalletSDKError(
      code: json['code'],
      category: json['category'],
      details: json['details'],
      traceID: json['traceID'],
    );
  }
}

class CredentialDisplayData {
  final String issuerName;
  final String overviewName;
  final String logo;
  final String backgroundColor;
  final String textColor;
  final List<CredentialDisplayClaim> claims;

//<editor-fold desc="Data Methods">
  const CredentialDisplayData({
    required this.issuerName,
    required this.overviewName,
    required this.logo,
    required this.backgroundColor,
    required this.textColor,
    required this.claims,
  });

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CredentialDisplayData &&
          runtimeType == other.runtimeType &&
          issuerName == other.issuerName &&
          overviewName == other.overviewName &&
          logo == other.logo &&
          backgroundColor == other.backgroundColor &&
          textColor == other.textColor &&
          claims == other.claims);

  @override
  int get hashCode =>
      issuerName.hashCode ^
      overviewName.hashCode ^
      logo.hashCode ^
      backgroundColor.hashCode ^
      textColor.hashCode ^
      claims.hashCode;

  @override
  String toString() {
    return 'CredentialDisplayData{' +
        ' issuerName: $issuerName,' +
        ' overviewName: $overviewName,' +
        ' logo: $logo,' +
        ' backgroundColor: $backgroundColor,' +
        ' textColor: $textColor,' +
        '}';
  }

  CredentialDisplayData copyWith({
    String? issuerName,
    String? overviewName,
    String? logo,
    String? backgroundColor,
    String? textColor,
    List<CredentialDisplayClaim>? claims,
  }) {
    return CredentialDisplayData(
      issuerName: issuerName ?? this.issuerName,
      overviewName: overviewName ?? this.overviewName,
      logo: logo ?? this.logo,
      backgroundColor: backgroundColor ?? this.backgroundColor,
      textColor: textColor ?? this.textColor,
      claims: claims ?? this.claims,
    );
  }

  Map<String, dynamic> toMap() {
    return {
      'issuerName': issuerName,
      'overviewName': overviewName,
      'logo': logo,
      'backgroundColor': backgroundColor,
      'textColor': textColor,
      'claims': claims.map((c) => c.toMap())
    };
  }

  factory CredentialDisplayData.fromMap(Map<String, dynamic> map) {
    List<dynamic> claims = map['claims'];

    return CredentialDisplayData(
        issuerName: map['issuerName'] as String,
        overviewName: map['overviewName'] as String,
        logo: map['logo'] as String,
        backgroundColor: map['backgroundColor'] as String,
        textColor: map['textColor'] as String,
        claims: claims.map((c) => CredentialDisplayClaim.fromMap(c.cast<String, dynamic>())).toList());
  }

//</editor-fold>
}

class CredentialDisplayClaim {
  final String rawValue;
  final String valueType;
  final String label;
  final String? value;
  final int? order;

//<editor-fold desc="Data Methods">
  const CredentialDisplayClaim({
    required this.rawValue,
    required this.valueType,
    required this.label,
    this.value,
    this.order,
  });

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CredentialDisplayClaim &&
          runtimeType == other.runtimeType &&
          rawValue == other.rawValue &&
          valueType == other.valueType &&
          label == other.label &&
          value == other.value &&
          order == other.order);

  @override
  int get hashCode => rawValue.hashCode ^ valueType.hashCode ^ label.hashCode ^ value.hashCode ^ order.hashCode;

  @override
  String toString() {
    return 'CredentialDisplayClaim{' +
        ' rawValue: $rawValue,' +
        ' valueType: $valueType,' +
        ' label: $label,' +
        ' value: $value,' +
        ' order: $order,' +
        '}';
  }

  CredentialDisplayClaim copyWith({
    String? rawValue,
    String? valueType,
    String? label,
    String? value,
    int? order,
  }) {
    return CredentialDisplayClaim(
      rawValue: rawValue ?? this.rawValue,
      valueType: valueType ?? this.valueType,
      label: label ?? this.label,
      value: value ?? this.value,
      order: order ?? this.order,
    );
  }

  Map<String, dynamic> toMap() {
    return {
      'rawValue': rawValue,
      'valueType': valueType,
      'label': label,
      'value': value,
      'order': order,
    };
  }

  factory CredentialDisplayClaim.fromMap(Map<String, dynamic> map) {
    return CredentialDisplayClaim(
      rawValue: map['rawValue'] as String,
      valueType: map['valueType'] as String,
      label: map['label'] as String,
      value: map['value'] as String?,
      order: map['order'] as int?,
    );
  }

//</editor-fold>
}

class VerifierDisplayData {
  final String name;
  final String did;
  final String purpose;
  final String logoURI;

//<editor-fold desc="Data Methods">
  const VerifierDisplayData({
    required this.name,
    required this.did,
    required this.purpose,
    required this.logoURI,
  });

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is VerifierDisplayData &&
          runtimeType == other.runtimeType &&
          name == other.name &&
          did == other.did &&
          purpose == other.purpose &&
          logoURI == other.logoURI);

  @override
  int get hashCode => name.hashCode ^ did.hashCode ^ purpose.hashCode ^ logoURI.hashCode;

  @override
  String toString() {
    return 'VerifierDisplayData{' +
        ' name: $name,' +
        ' did: $did,' +
        ' purpose: $purpose,' +
        ' logoURI: $logoURI,' +
        '}';
  }

  VerifierDisplayData copyWith({
    String? name,
    String? did,
    String? purpose,
    String? logoURI,
  }) {
    return VerifierDisplayData(
      name: name ?? this.name,
      did: did ?? this.did,
      purpose: purpose ?? this.purpose,
      logoURI: logoURI ?? this.logoURI,
    );
  }

  Map<String, dynamic> toMap() {
    return {
      'name': this.name,
      'did': this.did,
      'purpose': this.purpose,
      'logoURI': this.logoURI,
    };
  }

  factory VerifierDisplayData.fromMap(Map<String, dynamic> map) {
    return VerifierDisplayData(
      name: map['name'] as String,
      did: map['did'] as String,
      purpose: map['purpose'] as String,
      logoURI: map['logoURI'] as String,
    );
  }

//</editor-fold>
}