package com.volna.app.profile.presentation

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.config.AppConfig
import com.volna.app.core.logging.AppLogger
import com.volna.app.core.phone.formatPhoneNumber
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.ActionStatus
import com.volna.app.core.ui.Loadable
import com.volna.app.core.ui.PhoneNumberVisualTransformation
import com.volna.app.uikit.icons.ArrowRight
import com.volna.app.uikit.icons.Edit
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.VolnaIcon

@Composable
fun ProfileScreen(
    state: ProfileState,
    appConfig: AppConfig,
    onIntent: (ProfileIntent) -> Unit,
    modifier: Modifier = Modifier,
) {
    val snackbarHostState = remember { SnackbarHostState() }
    val uriHandler = LocalUriHandler.current
    val openExternalUrl: (String) -> Unit = { url ->
        runCatching { uriHandler.openUri(url) }
            .onFailure { failure -> AppLogger.e(failure, "Failed to open external URL: $url") }
    }

    LaunchedEffect(Unit) {
        onIntent(ProfileIntent.Load)
    }

    LaunchedEffect(state.message) {
        val message = state.message
        if (message != null) {
            snackbarHostState.showSnackbar(message)
            onIntent(ProfileIntent.MessageShown)
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        Text(
            text = "Профиль",
            modifier = Modifier
                .fillMaxWidth()
                .padding(26.dp),
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
        )
        Box(
            modifier = Modifier
                .fillMaxSize()
                .widthIn(max = VolnaTheme.tokens.sizing.screenMaxWidth),
        ) {
            when (val profile = state.profile) {
                Loadable.Initial,
                Loadable.Loading -> CircularProgressIndicator(
                    modifier = Modifier.align(Alignment.Center),
                )
                is Loadable.Content -> ProfileContent(
                    state = state,
                    appConfig = appConfig,
                    clientName = profile.value.name.orEmpty(),
                    phone = profile.value.phone.value,
                    onOpenExternalUrl = openExternalUrl,
                    onIntent = onIntent,
                )
                is Loadable.Error -> ProfileError(onRetry = { onIntent(ProfileIntent.Load) })
                is Loadable.Empty -> ProfileError(onRetry = { onIntent(ProfileIntent.Load) })
            }
            SnackbarHost(
                hostState = snackbarHostState,
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .padding(VolnaTheme.tokens.spacing.md),
            )
        }
    }

    if (state.logoutConfirmVisible) {
        LogoutConfirmDialog(
            onConfirm = { onIntent(ProfileIntent.LogoutConfirmed) },
            onDismiss = { onIntent(ProfileIntent.LogoutDismissed) },
        )
    }
    if (state.deleteConfirmVisible) {
        DeleteAccountConfirmDialog(
            onConfirm = { onIntent(ProfileIntent.DeleteConfirmed) },
            onDismiss = { onIntent(ProfileIntent.DeleteDismissed) },
        )
    }
}

@Composable
private fun ProfileContent(
    state: ProfileState,
    appConfig: AppConfig,
    clientName: String,
    phone: String,
    onOpenExternalUrl: (String) -> Unit,
    onIntent: (ProfileIntent) -> Unit,
) {
    Column(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .verticalScroll(rememberScrollState())
            .padding(start = VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.md),
    ) {
        when (state.mode) {
            ProfileMode.View -> ProfileViewContent(
                state = state,
                appConfig = appConfig,
                clientName = clientName,
                phone = phone,
                onOpenExternalUrl = onOpenExternalUrl,
                onIntent = onIntent,
            )
            ProfileMode.Edit -> ProfileEditContent(
                state = state,
                onIntent = onIntent,
            )
            ProfileMode.ConfirmPhone -> ProfilePhoneConfirmContent(
                state = state,
                onIntent = onIntent,
            )
        }
    }
}

@Composable
private fun ProfileViewContent(
    state: ProfileState,
    appConfig: AppConfig,
    clientName: String,
    phone: String,
    onOpenExternalUrl: (String) -> Unit,
    onIntent: (ProfileIntent) -> Unit,
) {
    ProfileInfoRow(
        label = null,
        value = clientName.ifBlank { "Имя" },
        placeholder = clientName.isBlank(),
        onClick = { onIntent(ProfileIntent.EditClicked) },
    )
    ProfileInfoRow(
        label = "Телефон",
        value = formatPhoneNumber(phone),
        onClick = { onIntent(ProfileIntent.EditClicked) },
    )
    Spacer(Modifier.height(VolnaTheme.tokens.spacing.md))
    ProfileLinks(
        appConfig = appConfig,
        onOpenExternalUrl = onOpenExternalUrl,
    )
    ProfileLogoutButton(state = state, onIntent = onIntent)
}

@Composable
private fun ProfileEditContent(
    state: ProfileState,
    onIntent: (ProfileIntent) -> Unit,
) {
    ProfileTextField(
        value = state.nameInput,
        onValueChange = { onIntent(ProfileIntent.NameChanged(it)) },
        label = "Имя",
        enabled = !state.isSubmitting,
    )
    ProfileTextField(
        value = state.phoneInput,
        onValueChange = { onIntent(ProfileIntent.PhoneChanged(it)) },
        label = "Телефон",
        enabled = !state.isSubmitting,
        keyboardType = KeyboardType.Phone,
        visualTransformation = PhoneNumberVisualTransformation(),
    )
    state.fieldError?.let {
        Text(it, color = MaterialTheme.colorScheme.error)
    }
    Button(
        onClick = { onIntent(ProfileIntent.SaveClicked) },
        enabled = state.canSave,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight),
    ) {
        Text(if (state.isSubmitting) "Сохраняем..." else "Сохранить", fontWeight = FontWeight.Bold)
    }
    OutlinedButton(
        onClick = { onIntent(ProfileIntent.EditCancelled) },
        enabled = !state.isSubmitting,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight),
    ) {
        Text("Отменить")
    }
}

@Composable
private fun ProfilePhoneConfirmContent(
    state: ProfileState,
    onIntent: (ProfileIntent) -> Unit,
) {
    Text(
        text = "Подтвердите новый номер кодом из SMS",
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Bold,
    )
    Text(
        text = formatPhoneNumber(state.pendingPhone ?: state.phoneInput),
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    ProfileTextField(
        value = state.codeInput,
        onValueChange = { onIntent(ProfileIntent.CodeChanged(it)) },
        label = "Код из SMS",
        enabled = !state.isSubmitting,
    )
    state.fieldError?.let {
        Text(it, color = MaterialTheme.colorScheme.error)
    }
    Button(
        onClick = { onIntent(ProfileIntent.ConfirmPhoneClicked) },
        enabled = state.canConfirmPhone,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight),
    ) {
        Text(if (state.isSubmitting) "Проверяем..." else "Подтвердить", fontWeight = FontWeight.Bold)
    }
    TextButton(
        onClick = { onIntent(ProfileIntent.ResendPhoneCode) },
        enabled = state.canResendCode,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            if (state.resendSecondsRemaining > 0) {
                "Отправить код повторно (00:${state.resendSecondsRemaining.toString().padStart(2, '0')})"
            } else {
                "Отправить код повторно"
            },
        )
    }
    OutlinedButton(
        onClick = { onIntent(ProfileIntent.BackToEdit) },
        enabled = !state.isSubmitting,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight),
    ) {
        Text("Назад к редактированию")
    }
}

@Composable
private fun ProfileTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    enabled: Boolean,
    keyboardType: KeyboardType = KeyboardType.Text,
    visualTransformation: VisualTransformation = VisualTransformation.None,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        enabled = enabled,
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
        visualTransformation = visualTransformation,
        label = { Text(label) },
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun ProfileInfoRow(
    label: String?,
    value: String,
    placeholder: Boolean = false,
    onClick: (() -> Unit)? = null,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight)
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .then(if (onClick != null) Modifier.clickable { onClick() } else Modifier)
            .padding(horizontal = VolnaTheme.tokens.spacing.md),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(
            verticalArrangement = Arrangement.Center,
        ) {
            if (label != null) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF797979),
                )
            }
            Text(
                text = value,
                style = MaterialTheme.typography.bodyLarge,
                color = if (placeholder) Color(0xFF797979) else MaterialTheme.colorScheme.onSurface,
            )
        }
        VolnaIcon(
            imageVector = Icons.Edit,
            contentDescription = "Редактировать",
            tint = MaterialTheme.colorScheme.onSurface,
            size = VolnaTheme.tokens.spacing.lg,
        )
    }
}

@Composable
private fun ProfileLinks(
    appConfig: AppConfig,
    onOpenExternalUrl: (String) -> Unit,
) {
    // SCR-007 / AC-009: links are opened only when provided by app configuration.
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(Color.Transparent)
            .padding(top = VolnaTheme.tokens.spacing.sm),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(Color(0xFFF2F2F2)),
        )
        InfoLine(
            label = "Правила клуба",
            value = if (appConfig.rulesUrl != null) null else "не настроено",
            onClick = appConfig.rulesUrl?.let { url -> { onOpenExternalUrl(url) } },
        )
        InfoLine(
            label = "Поддержка",
            value = if (appConfig.supportUrl != null) null else "не настроено",
            onClick = appConfig.supportUrl?.let { url -> { onOpenExternalUrl(url) } },
        )
        InfoLine("Версия приложения", appConfig.appVersion)
    }
}

@Composable
private fun ProfileLogoutButton(
    state: ProfileState,
    onIntent: (ProfileIntent) -> Unit,
) {
    Button(
        onClick = { onIntent(ProfileIntent.LogoutClicked) },
        enabled = state.actionStatus == ActionStatus.Idle,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        colors = ButtonDefaults.outlinedButtonColors(
            containerColor = MaterialTheme.colorScheme.surface,
            contentColor = MaterialTheme.colorScheme.primary,
            disabledContainerColor = MaterialTheme.colorScheme.surface,
            disabledContentColor = MaterialTheme.colorScheme.onSurfaceVariant,
        ),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.primary),
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight),
    ) {
        Text(if (state.isSubmitting) "Выходим..." else "Выйти", fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun InfoLine(
    label: String,
    value: String?,
    onClick: (() -> Unit)? = null,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (onClick != null) Modifier.clickable { onClick() } else Modifier),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = Color(0xFF797979),
        )
        if (value != null) {
            Text(
                text = value,
                style = MaterialTheme.typography.bodyMedium,
                color = Color(0xFF797979),
            )
        } else {
            VolnaIcon(
                imageVector = Icons.ArrowRight,
                contentDescription = null,
                tint = Color(0xFF797979),
                size = 16.dp,
            )
        }
    }
}

@Composable
private fun ProfileError(
    onRetry: () -> Unit,
) {
    Column(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .padding(start = VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Не удалось загрузить профиль", textAlign = TextAlign.Center)
        Button(onClick = onRetry) {
            Text("Повторить")
        }
    }
}

@Composable
private fun LogoutConfirmDialog(
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Выйти из аккаунта?") },
        text = { Text("После выхода для записи на заезд нужно будет снова ввести телефон и код.") },
        confirmButton = {
            TextButton(onClick = onConfirm) {
                Text("Выйти")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Не выходить")
            }
        },
    )
}

@Composable
private fun DeleteAccountConfirmDialog(
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Удалить аккаунт?") },
        text = { Text("Профиль будет удалён. Для новых записей потребуется зарегистрироваться снова.") },
        confirmButton = {
            TextButton(onClick = onConfirm) {
                Text("Удалить", color = MaterialTheme.colorScheme.error)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Отменить")
            }
        },
    )
}
