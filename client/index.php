<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Herb | Search The Indie Web</title>
</head>

<body>
    <h1>Herb</h1>

    <form action="search" method="get">
        <input type="text" maxlength="255" required>
        <input type="submit" value="Go">
    </form>

    <footer>
        Happy <?php echo strtolower(date('l', time())); ?>!
    </footer>
</body>


</html>