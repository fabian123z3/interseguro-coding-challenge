# Interseguro — Coding Challenge

**División TI — Junio 2024**

---

## 1. Descripción del Desafío Técnico

### Consideraciones técnicas
- **Lenguajes:** Utilizar el lenguaje de programación **Go (Golang)** para una API y **Node.js** para la otra API.
- **Frameworks:** Implementar la solución utilizando los frameworks **Fiber** para la API en Go y **Express.js** para la API en Node.js.
- **Documentación:** Documentar el código de manera clara y concisa, siguiendo las mejores prácticas de codificación.
- **Contenedores:** Utilizar **Docker** para contenerizar las aplicaciones y facilitar su despliegue en diferentes entornos.
- **Comunicación entre APIs:** Implementar la comunicación entre las dos APIs utilizando un mecanismo como **HTTP**.
- **Nube:** Utilizar servicios en la nube para la implementación y el despliegue de las aplicaciones.

### Arquitectura de la solución
- **API en Go:** Recibirá la matriz original como entrada, realizará la rotación de la matriz y/o factorización QR, y enviará los datos resultantes a la segunda API en Node.js.
- **API en Node.js:** Recibirá los datos de las matrices de la API en Go, calculará estadísticas sobre los datos y devolverá estas estadísticas como resultado.

---

## 2. Funcionalidad

### Funcionalidad requerida
1. **Crear dos APIs RESTful:**
   - **API en Go:** Recibe como entrada un array de arrays de números que representa una matriz rectangular y devuelve la factorización QR de dicha matriz.
   - **API en Node.js:** Recibe el resultado de las matrices devueltas por la primera API y realiza una operación adicional sobre los datos (ver sección de estadísticas).
2. **Eficiencia y corrección:** Implementar la lógica para realizar la rotación de la matriz y la operación adicional de manera eficiente y correcta en cada API.

### Funcionalidad opcional (Cumplida al 100%)
- ✅ **Frontend:** Implementar un frontend que consuma ambas APIs y muestre los resultados de la rotación de la matriz y las estadísticas adicionales.
- ✅ **Seguridad JWT:** Aplicar un nivel de seguridad utilizando JWT para proteger las consultas a las APIs.
- ✅ **Pruebas:** Implementar pruebas unitarias y de integración para garantizar la calidad del código en ambas APIs.

---

## 3. Operación Adicional (Estadísticas en Node.js)

La segunda API calculará lo siguiente sobre los datos de las matrices devueltas:
- **Valor máximo:** El valor máximo encontrado en las matrices.
- **Valor mínimo:** El valor mínimo encontrado en las matrices.
- **Promedio:** El promedio de todos los valores de las matrices.
- **Suma total:** La suma total de todos los valores de las matrices.
- **Matriz diagonal:** Verificar si alguna matriz es diagonal.

---

## 4. Consideraciones de Evaluación
- No hay un estándar específico para los nombres de los objetos creados, pero se espera coherencia en su estructura y documentación.
- En caso de dudas en el enunciado, se espera que el candidato tome decisiones informadas y las sustente durante la entrevista.
- Se valorará la eficiencia y la elegancia de la solución implementada, así como la capacidad del candidato para comunicar y defender sus decisiones técnicas.
