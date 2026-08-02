package com.volna.app.core.ui

import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.input.OffsetMapping
import androidx.compose.ui.text.input.TransformedText
import androidx.compose.ui.text.input.VisualTransformation
import com.volna.app.core.phone.formatPhoneNumberWithPositions

class PhoneNumberVisualTransformation : VisualTransformation {
    override fun filter(text: AnnotatedString): TransformedText {
        val result = formatPhoneNumberWithPositions(text.text)

        val offsetMapping = object : OffsetMapping {
            override fun originalToTransformed(offset: Int): Int {
                if (offset <= 0) return 0
                val positions = result.digitPositions
                if (positions.isEmpty()) return 0
                if (offset >= positions.size) return result.text.length
                return positions[offset - 1] + 1
            }

            override fun transformedToOriginal(offset: Int): Int {
                val positions = result.digitPositions
                if (positions.isEmpty()) return 0
                var count = 0
                while (count < positions.size && positions[count] < offset) {
                    count++
                }
                return count
            }
        }

        return TransformedText(AnnotatedString(result.text), offsetMapping)
    }
}
